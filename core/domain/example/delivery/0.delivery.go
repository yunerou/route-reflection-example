package rawdocdel

import (
	"context"
	"encoding/json"
)

type ExampleHandler interface {
	ExampleHandlerFunc(ctx context.Context, reqParam ExampleReqParam, reqBody ExampleReqBody) (ExampleRespBody, error)
}

type ExampleReqParam struct {
	// Documentation for Param1: this is param1
	Param1 string `path:"param1"`
	Param2 string `path:"param2"`
	// Documentation for Param2: this is param2
	Param3 int `query:"param3"`
}

type ExampleReqBody struct {
	// Documentation for Field1: this is field1
	Field1 string `json:"field1"`
	// Documentation for Field2: this is field2
	Field2 int `json:"field2"`
}

type ExampleRespBody struct {
	// Documentation for Message: this is result
	Message       string          `json:"msg"`
	RequestParams json.RawMessage `json:"request_params"`
	RequestBody   json.RawMessage `json:"request_body"`
}
