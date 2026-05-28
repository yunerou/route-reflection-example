package delivery

import (
	"context"
	"encoding/json"

	exampleDeli "github.com/yunerou/niarb/core/domain/example/delivery"
)

func (d *deli) ExampleHandlerFunc(ctx context.Context, reqParam exampleDeli.ExampleReqParam, reqBody exampleDeli.ExampleReqBody) (exampleDeli.ExampleRespBody, error) {
	requestParams, _ := json.MarshalIndent(reqParam, "", "  ")
	requestBody, _ := json.MarshalIndent(reqBody, "", "  ")
	return exampleDeli.ExampleRespBody{
		Message:       "This is example response",
		RequestParams: requestParams,
		RequestBody:   requestBody,
	}, nil
}
