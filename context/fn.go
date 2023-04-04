package context

import (
	"encoding/json"
	"fmt"
)

func toJSONBuffer(args ...any) ([]byte, error) {
	data := args[0]
	switch args[0].(type) {
	case string:
		jsonStr := fmt.Sprintf(data.(string), args[1:]...)
		data = json.RawMessage(jsonStr)
	}

	return json.Marshal(&data)
}

func toJSONP(jsonStr, callback string) string {
	return fmt.Sprintf("/**/ typeof %v === 'function' && %v(%v);", callback, callback, jsonStr)
}
