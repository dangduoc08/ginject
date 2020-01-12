package router

import (
	"encoding/json"
	"fmt"
)

// Implements ResponseExtender interface
func (res *response) Status(statusCode int) ResponseExtender {
	res.WriteHeader(statusCode)
	return res
}

func (res *response) Type(contentType string) ResponseExtender {
	res.Header().Set("Content-Type", contentType)
	return res
}

func (res *response) Send(content string, arguments ...interface{}) ResponseExtender {
	res.Type("text/html")
	fmt.Fprintf(res.ResponseWriter, content, arguments...)
	return res
}

func (res *response) JSON(datas ...interface{}) ResponseExtender {
	data := datas[0]
	switch data.(type) {
	case string:
		data = json.RawMessage(data.(string))
	default:
	}
	buffer, err := json.Marshal(&data)
	if err != nil {
		fmt.Println(err)
	}
	res.Type("application/json")
	res.Write(buffer)
	return res
}
