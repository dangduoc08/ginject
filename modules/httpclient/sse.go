package httpclient

import (
	"bufio"
	"io"
	"strconv"
	"strings"
)

// SSEEvent represents a single Server-Sent Event.
type SSEEvent struct {
	ID    string
	Event string
	Data  string
	Retry int
}

// SSEReader parses Server-Sent Events from a stream.
type SSEReader struct {
	scanner *bufio.Scanner
}

// NewSSEReader creates an SSEReader that reads events from r.
func NewSSEReader(r io.Reader) *SSEReader {
	return &SSEReader{scanner: bufio.NewScanner(r)}
}

// Next blocks until the next complete SSE event is available.
// Returns (nil, false) when the stream ends or an error occurs.
func (sr *SSEReader) Next() (*SSEEvent, bool) {
	evt := &SSEEvent{}
	var dataLines []string
	hasData := false

	for sr.scanner.Scan() {
		line := sr.scanner.Text()
		if line == "" {
			if hasData {
				evt.Data = strings.Join(dataLines, "\n")
				return evt, true
			}
			evt = &SSEEvent{}
			dataLines = dataLines[:0]
			continue
		}
		if strings.HasPrefix(line, ":") {
			continue
		}
		field, value, _ := strings.Cut(line, ":")
		value = strings.TrimPrefix(value, " ")
		switch field {
		case "id":
			evt.ID = value
		case "event":
			evt.Event = value
		case "data":
			dataLines = append(dataLines, value)
			hasData = true
		case "retry":
			if n, err := strconv.Atoi(value); err == nil {
				evt.Retry = n
			}
		}
	}
	return nil, false
}
