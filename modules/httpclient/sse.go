package httpclient

import (
	"bufio"
	"io"
	"strconv"
	"strings"
)

type SSEEvent struct {
	ID    string
	Event string
	Data  string
	Retry int
}

type SSEReader struct {
	scanner *bufio.Scanner
}

func NewSSEReader(r io.Reader) *SSEReader {
	return &SSEReader{scanner: bufio.NewScanner(r)}
}

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
