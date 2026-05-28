package constants

import (
	"os"
	"path/filepath"
	"runtime"
)

var (
	projectPath string
)

func ProjectPath() string {
	if projectPath != "" {
		return projectPath
	}
	_, b, _, _ := runtime.Caller(0)
	rootPath := filepath.Join(filepath.Dir(b), "./../..")
	return rootPath
}

func ConfigPath() string {
	if os.Getenv("APP_CONFIG_PATH") != "" {
		return os.Getenv("APP_CONFIG_PATH")
	}
	return filepath.Join(ProjectPath(), "config")
}
