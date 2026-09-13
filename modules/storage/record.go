package storage

import (
	"encoding/binary"
	"hash/crc32"
)

type recType uint8

const (
	recInsert     recType = 1
	recUpdate     recType = 2
	recDelete     recType = 3
	recTxBegin    recType = 4
	recTxCommit   recType = 5
	recTxRollback recType = 6
)

type record struct {
	rtype     recType
	txID      uint64
	table     string
	id        string
	timestamp int64
	payload   []byte
}

func encodeRecord(r record) []byte {
	tbl := []byte(r.table)
	id := []byte(r.id)

	innerSize := 1 + 8 + 2 + len(tbl) + 2 + len(id) + 8 + 4 + len(r.payload)
	buf := make([]byte, 8+innerSize)

	off := 8
	buf[off] = byte(r.rtype)
	off++
	binary.LittleEndian.PutUint64(buf[off:], r.txID)
	off += 8
	binary.LittleEndian.PutUint16(buf[off:], uint16(len(tbl)))
	off += 2
	copy(buf[off:], tbl)
	off += len(tbl)
	binary.LittleEndian.PutUint16(buf[off:], uint16(len(id)))
	off += 2
	copy(buf[off:], id)
	off += len(id)
	binary.LittleEndian.PutUint64(buf[off:], uint64(r.timestamp))
	off += 8
	binary.LittleEndian.PutUint32(buf[off:], uint32(len(r.payload)))
	off += 4
	copy(buf[off:], r.payload)

	binary.LittleEndian.PutUint32(buf[0:], uint32(4+innerSize))
	binary.LittleEndian.PutUint32(buf[4:], crc32.ChecksumIEEE(buf[8:]))
	return buf
}

func decodeRecord(data []byte) (record, int, error) {
	if len(data) < 8 {
		return record{}, 0, ErrCorrupt
	}
	totalSize := int(binary.LittleEndian.Uint32(data[0:]))
	full := 4 + totalSize
	if len(data) < full {
		return record{}, 0, ErrCorrupt
	}
	crc := binary.LittleEndian.Uint32(data[4:])
	if crc32.ChecksumIEEE(data[8:full]) != crc {
		return record{}, 0, ErrCorrupt
	}

	off := 8
	if full <= off {
		return record{}, 0, ErrCorrupt
	}
	rt := recType(data[off])
	off++

	if off+8 > full {
		return record{}, 0, ErrCorrupt
	}
	txID := binary.LittleEndian.Uint64(data[off:])
	off += 8

	if off+2 > full {
		return record{}, 0, ErrCorrupt
	}
	tlen := int(binary.LittleEndian.Uint16(data[off:]))
	off += 2
	if off+tlen > full {
		return record{}, 0, ErrCorrupt
	}
	tbl := string(data[off : off+tlen])
	off += tlen

	if off+2 > full {
		return record{}, 0, ErrCorrupt
	}
	ilen := int(binary.LittleEndian.Uint16(data[off:]))
	off += 2
	if off+ilen > full {
		return record{}, 0, ErrCorrupt
	}
	id := string(data[off : off+ilen])
	off += ilen

	if off+8+4 > full {
		return record{}, 0, ErrCorrupt
	}
	ts := int64(binary.LittleEndian.Uint64(data[off:]))
	off += 8
	plen := int(binary.LittleEndian.Uint32(data[off:]))
	off += 4
	if off+plen > full {
		return record{}, 0, ErrCorrupt
	}
	payload := make([]byte, plen)
	copy(payload, data[off:off+plen])

	return record{
		rtype: rt, txID: txID, table: tbl, id: id,
		timestamp: ts, payload: payload,
	}, full, nil
}
