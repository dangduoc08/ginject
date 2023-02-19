package ctx

import (
	"encoding/json"
	"fmt"
)

func handleJSON(args ...any) ([]byte, error) {
	data := args[0]
	switch args[0].(type) {
	case string:
		str := fmt.Sprintf(data.(string), args[1:]...)
		data = json.RawMessage(str)
	}

	return json.Marshal(&data)
}

func buildJSONP(jsonStr, cb string) string {
	return fmt.Sprintf("/**/ typeof %v === 'function' && %v(%v);", cb, cb, jsonStr)
}
