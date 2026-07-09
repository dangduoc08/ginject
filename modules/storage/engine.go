package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
)

const maxSegSize = 64 << 20 // 64 MB per segment

// location describes where a record lives on disk.
type location struct {
	segID  int
	offset int64
	size   int // total bytes on disk (including 4-byte size prefix)
}

type segFile struct {
	id   int
	f    *os.File
	size int64
}

// engine manages append-only segment files for a single table.
// mu protects the segment list, current segment pointer, and all indexes.
type engine struct {
	mu      sync.RWMutex
	dir     string
	segs    []*segFile
	current *segFile
	nextSeg int
	idx     *tableIndex
}

func openEngine(dir string) (*engine, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	e := &engine{dir: dir, idx: newTableIndex()}
	if err := e.loadSegments(); err != nil {
		return nil, err
	}
	if err := e.rebuildIndex(); err != nil {
		return nil, err
	}
	return e, nil
}

func (e *engine) loadSegments() error {
	entries, err := os.ReadDir(e.dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasPrefix(name, "seg_") || !strings.HasSuffix(name, ".db") {
			continue
		}
		idStr := strings.TrimSuffix(strings.TrimPrefix(name, "seg_"), ".db")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			continue
		}
		path := filepath.Join(e.dir, name)
		f, err := os.OpenFile(path, os.O_RDWR|os.O_APPEND, 0o644)
		if err != nil {
			return err
		}
		info, err := f.Stat()
		if err != nil {
			_ = f.Close()
			return err
		}
		e.segs = append(e.segs, &segFile{id: id, f: f, size: info.Size()})
		if id >= e.nextSeg {
			e.nextSeg = id + 1
		}
	}
	sort.Slice(e.segs, func(i, j int) bool { return e.segs[i].id < e.segs[j].id })

	if len(e.segs) > 0 {
		last := e.segs[len(e.segs)-1]
		if last.size < maxSegSize {
			e.current = last
		}
	}
	if e.current == nil {
		return e.newSegment()
	}
	return nil
}

func (e *engine) newSegment() error {
	id := e.nextSeg
	e.nextSeg++
	path := filepath.Join(e.dir, fmt.Sprintf("seg_%07d.db", id))
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	seg := &segFile{id: id, f: f}
	e.segs = append(e.segs, seg)
	e.current = seg
	return nil
}

// rebuildIndex rebuilds the in-memory index by scanning all segments.
// Phase 1: determine the final live set (accounting for transactions).
// Phase 2: load each live record and build secondary/text indexes.
func (e *engine) rebuildIndex() error {
	type txEntry struct {
		r   record
		loc location
	}
	type txBuf struct {
		latestByID map[string]txEntry // id → latest op entry within this tx
	}

	primary := make(map[string]location) // live id → latest location
	txs := make(map[uint64]*txBuf)

	apply := func(r record, loc location) {
		switch r.rtype {
		case recInsert, recUpdate:
			primary[r.id] = loc
		case recDelete:
			delete(primary, r.id)
		}
	}

	for _, seg := range e.segs {
		if err := scanSegFile(seg, func(off int64, r record, size int) error {
			loc := location{segID: seg.id, offset: off, size: size}
			switch r.rtype {
			case recTxBegin:
				txs[r.txID] = &txBuf{latestByID: make(map[string]txEntry)}
			case recTxCommit:
				if buf, ok := txs[r.txID]; ok {
					for _, entry := range buf.latestByID {
						apply(entry.r, entry.loc)
					}
					delete(txs, r.txID)
				}
			case recTxRollback:
				delete(txs, r.txID)
			default:
				if r.txID != 0 {
					if buf, ok := txs[r.txID]; ok {
						buf.latestByID[r.id] = txEntry{r: r, loc: loc}
					}
				} else {
					apply(r, loc)
				}
			}
			return nil
		}); err != nil {
			return err
		}
	}

	// Phase 2: build secondary and text indexes from live records
	for id, loc := range primary {
		seg := e.segByID(loc.segID)
		if seg == nil {
			continue
		}
		r, err := readRecordAt(seg, loc)
		if err != nil {
			continue
		}
		e.idx.setPrimary(id, loc)
		if len(r.payload) > 0 {
			data, _, _, err := unmarshalPayload(r.payload)
			if err == nil {
				e.idx.updateSecondary(id, nil, data)
				e.idx.updateText(id, nil, data)
			}
		}
	}
	return nil
}

// writeRecord appends a record to the current segment.
// Caller must hold the write lock.
func (e *engine) writeRecord(r record) (location, error) {
	data := encodeRecord(r)
	if e.current.size+int64(len(data)) > maxSegSize && e.current.size > 0 {
		if err := e.newSegment(); err != nil {
			return location{}, err
		}
	}
	offset := e.current.size
	n, err := e.current.f.Write(data)
	if err != nil {
		return location{}, err
	}
	e.current.size += int64(n)
	return location{segID: e.current.id, offset: offset, size: len(data)}, nil
}

// readDoc reads and parses the document at the given location.
// Caller must hold at least a read lock.
func (e *engine) readDoc(id string, loc location) (Document, error) {
	seg := e.segByID(loc.segID)
	if seg == nil {
		return Document{}, ErrNotFound
	}
	r, err := readRecordAt(seg, loc)
	if err != nil {
		return Document{}, err
	}
	data, createdAt, updatedAt, err := unmarshalPayload(r.payload)
	if err != nil {
		return Document{}, ErrCorrupt
	}
	return Document{ID: id, Data: data, CreatedAt: createdAt, UpdatedAt: updatedAt}, nil
}

func (e *engine) segByID(id int) *segFile {
	for _, s := range e.segs {
		if s.id == id {
			return s
		}
	}
	return nil
}

func readRecordAt(seg *segFile, loc location) (record, error) {
	buf := make([]byte, loc.size)
	if _, err := seg.f.ReadAt(buf, loc.offset); err != nil {
		return record{}, err
	}
	r, _, err := decodeRecord(buf)
	return r, err
}

func scanSegFile(seg *segFile, fn func(offset int64, r record, size int) error) error {
	if seg.size == 0 {
		return nil
	}
	data := make([]byte, seg.size)
	if _, err := seg.f.ReadAt(data, 0); err != nil {
		return err
	}
	off := 0
	for off < len(data) {
		r, n, err := decodeRecord(data[off:])
		if err != nil {
			// corrupted tail record — stop scanning this segment
			break
		}
		if err := fn(int64(off), r, n); err != nil {
			return err
		}
		off += n
	}
	return nil
}

func (e *engine) flush() error {
	e.mu.RLock()
	cur := e.current
	e.mu.RUnlock()
	if cur == nil {
		return nil
	}
	return cur.f.Sync()
}

func (e *engine) close() {
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, s := range e.segs {
		_ = s.f.Close()
	}
}

// compact rewrites all live records to a fresh set of segment files.
// Holds the write lock throughout; briefly blocks all reads and writes.
func (e *engine) compact() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// collect final live state
	primary := make(map[string]location)
	for _, seg := range e.segs {
		_ = scanSegFile(seg, func(off int64, r record, size int) error {
			loc := location{segID: seg.id, offset: off, size: size}
			switch r.rtype {
			case recInsert, recUpdate:
				if r.txID == 0 {
					primary[r.id] = loc
				}
			case recDelete:
				if r.txID == 0 {
					delete(primary, r.id)
				}
			}
			return nil
		})
	}

	tmpDir := filepath.Join(e.dir, "_compact")
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		return err
	}

	var newSegs []*segFile
	var curSeg *segFile
	nextID := 0

	rotateSeg := func() error {
		path := filepath.Join(tmpDir, fmt.Sprintf("seg_%07d.db", nextID))
		f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o644)
		if err != nil {
			return err
		}
		curSeg = &segFile{id: nextID, f: f}
		newSegs = append(newSegs, curSeg)
		nextID++
		return nil
	}
	if err := rotateSeg(); err != nil {
		_ = os.RemoveAll(tmpDir)
		return err
	}

	for id, loc := range primary {
		seg := e.segByID(loc.segID)
		if seg == nil {
			continue
		}
		r, err := readRecordAt(seg, loc)
		if err != nil {
			continue
		}
		r.id = id
		data := encodeRecord(r)
		if curSeg.size+int64(len(data)) > maxSegSize && curSeg.size > 0 {
			if err := rotateSeg(); err != nil {
				_ = os.RemoveAll(tmpDir)
				return err
			}
		}
		if _, err := curSeg.f.Write(data); err != nil {
			_ = os.RemoveAll(tmpDir)
			return err
		}
		curSeg.size += int64(len(data))
	}

	// move compacted segments into the table dir
	for i, s := range newSegs {
		_ = s.f.Close()
		dest := filepath.Join(e.dir, fmt.Sprintf("seg_%07d.db", i))
		if err := os.Rename(s.f.Name(), dest); err != nil {
			_ = os.RemoveAll(tmpDir)
			return err
		}
		f, err := os.OpenFile(dest, os.O_RDWR|os.O_APPEND, 0o644)
		if err != nil {
			_ = os.RemoveAll(tmpDir)
			return err
		}
		info, _ := f.Stat()
		newSegs[i] = &segFile{id: i, f: f, size: info.Size()}
	}
	_ = os.RemoveAll(tmpDir)

	// remove old segment files beyond the new count
	for j := len(newSegs); j < len(e.segs); j++ {
		_ = e.segs[j].f.Close()
		_ = os.Remove(e.segs[j].f.Name())
	}
	for j := 0; j < len(e.segs) && j < len(newSegs); j++ {
		_ = e.segs[j].f.Close()
	}

	e.segs = newSegs
	e.nextSeg = len(newSegs)
	if len(newSegs) > 0 {
		e.current = newSegs[len(newSegs)-1]
	} else {
		e.current = nil
		_ = e.newSegment()
	}

	// rebuild index from new segments
	e.idx = newTableIndex()
	_ = e.rebuildIndex()
	return nil
}
