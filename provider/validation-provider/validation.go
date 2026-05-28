package validationprovider

import (
	"github.com/samber/do/v2"
	"github.com/yunerou/niarb/pkg/validation"
)

func NewDI(i do.Injector) (validation.ValidationProvider, error) {
	validationIns := validation.NewValidationProvider(nil, nil, nil)

	return validationIns, nil
}
