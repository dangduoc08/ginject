package global

import (
	"encoding/json"
	"net/http"

	"github.com/dangduoc08/gooh"
	"github.com/dangduoc08/gooh/common"
	"github.com/dangduoc08/gooh/exception"
	"github.com/dangduoc08/gooh/modules/config"
)

type LoggingInterceptor struct {
	ConfigService config.ConfigService
	Logger        common.Logger
}

func (i LoggingInterceptor) Intercept(c gooh.Context, aggregation gooh.Aggregation) any {
	datas := []any{}

	if c.Method == http.MethodPost {
		body := c.Body()
		if len(body) > 0 {
			bodyJSON, _ := json.Marshal(body)
			bodyJSONStr := string(bodyJSON)
			datas = append(datas, "body", bodyJSONStr)
		} else {
			formMap := c.Form()
			if len(formMap) > 0 {
				formJSON, _ := json.Marshal(formMap)
				formJSONStr := string(formJSON)
				datas = append(datas, "body", formJSONStr)
			} else {
				datas = append(datas, "body", nil)
			}
		}
	}

	queryMap := c.Query()
	if len(queryMap) > 0 {
		queryJSON, _ := json.Marshal(queryMap)
		queryJSONStr := string(queryJSON)
		datas = append(datas, "query", queryJSONStr)
	} else {
		datas = append(datas, "query", nil)
	}

	paramMap := c.Param()
	if len(paramMap) > 0 {
		paramJSON, _ := json.Marshal(c.Param())
		paramJSONStr := string(paramJSON)
		datas = append(datas, "param", paramJSONStr)
	} else {
		datas = append(datas, "param", nil)
	}

	i.Logger.Debug(
		"Request",
		datas...,
	)

	return aggregation.Pipe(
		aggregation.Consume(func(data any) any {

			resJSON, _ := json.Marshal(data)
			resJSONStr := string(resJSON)
			if resJSONStr != "{}" {
				i.Logger.Debug("Success", "data", resJSONStr)
			} else {
				i.Logger.Debug("Success", "data", nil)
			}
			return data
		}),
		aggregation.Error(func(err any) any {
			if httpException, ok := err.(exception.HTTPException); ok {
				i.Logger.Debug("Error", "data", err, "message", httpException.GetResponse())
			} else {
				i.Logger.Debug("Error", "data", err)
			}

			return err
		}),
	)
}
