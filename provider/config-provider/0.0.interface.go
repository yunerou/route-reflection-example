package configprovider

import (
	"github.com/yunerou/niarb/shared/constants"
)

type globalEnvT struct {
	EnvType
	Version string
}

func (g *globalEnvT) LoadDefault() {
	g.EnvType.LoadDefault()
	g.Version = constants.Version
}

func (g *globalEnvT) Validate() {
	g.EnvType.Validate()
}

// ConfigStore provides access to application configuration
type ConfigStore interface {
	// Env returns the global environment configuration
	Env() *globalEnvT
}
