package httpclient

import (
	"context"
	"io"
	"os"
)

// Progress reports the current state of a streaming transfer.
type Progress struct {
	Total   int64
	Current int64
	Percent float64
}

type progressReader struct {
	r          io.Reader
	total      int64
	current    int64
	onProgress func(Progress)
}

func (pr *progressReader) Read(p []byte) (n int, err error) {
	n, err = pr.r.Read(p)
	if n > 0 {
		pr.current += int64(n)
		pct := float64(0)
		if pr.total > 0 {
			pct = float64(pr.current) / float64(pr.total) * 100
		}
		pr.onProgress(Progress{Total: pr.total, Current: pr.current, Percent: pct})
	}
	return
}

type progressReadCloser struct {
	io.ReadCloser
	total      int64
	current    int64
	onProgress func(Progress)
}

func (p *progressReadCloser) Read(buf []byte) (n int, err error) {
	n, err = p.ReadCloser.Read(buf)
	if n > 0 {
		p.current += int64(n)
		pct := float64(0)
		if p.total > 0 {
			pct = float64(p.current) / float64(p.total) * 100
		}
		p.onProgress(Progress{Total: p.total, Current: p.current, Percent: pct})
	}
	return
}

// cancelReadCloser calls cancel when the stream is closed, propagating context
// cancellation to the underlying transport after the caller is done reading.
type cancelReadCloser struct {
	io.ReadCloser
	cancel context.CancelFunc
}

func (c *cancelReadCloser) Close() error {
	c.cancel()
	return c.ReadCloser.Close()
}

func saveToFile(r io.Reader, dst string) error {
	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	_, err = io.Copy(f, r)
	return err
}
