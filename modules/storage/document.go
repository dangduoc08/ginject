package storage

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"time"
)

// Document is a stored record returned to callers.
type Document struct {
	ID        string
	Data      map[string]any
	CreatedAt time.Time
	UpdatedAt time.Time
}

// docPayload is what we store as JSON in the record payload.
type docPayload struct {
	C int64          `json:"c"` // createdAt unix nano
	U int64          `json:"u"` // updatedAt unix nano
	D map[string]any `json:"d"` // user data
}

func marshalPayload(data map[string]any, createdAt, updatedAt time.Time) ([]byte, error) {
	return json.Marshal(docPayload{
		C: createdAt.UnixNano(),
		U: updatedAt.UnixNano(),
		D: data,
	})
}

func unmarshalPayload(b []byte) (map[string]any, time.Time, time.Time, error) {
	var p docPayload
	if err := json.Unmarshal(b, &p); err != nil {
		return nil, time.Time{}, time.Time{}, err
	}
	return p.D, time.Unix(0, p.C), time.Unix(0, p.U), nil
}

func newID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
