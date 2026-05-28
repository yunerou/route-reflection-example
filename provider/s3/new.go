package s3provider

import (
	"github.com/samber/do/v2"

	configprovider "github.com/yunerou/niarb/provider/config-provider"
)

func NewDI(i do.Injector) (ObjectStorage, error) {
	cfg := do.MustInvoke[configprovider.ConfigStore](i)
	return New(cfg.Env().S3), nil
}
