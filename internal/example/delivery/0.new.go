package delivery

import (
	"github.com/samber/do/v2"
	exampleDeli "github.com/yunerou/niarb/core/domain/example/delivery"
)

type deli struct{}

func NewDI(i do.Injector) (exampleDeli.ExampleHandler, error) {
	return &deli{}, nil
}
