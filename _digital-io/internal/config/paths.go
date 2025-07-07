package config

import (
	"os"
	"path/filepath"
)

// GetExecutableDir returns the directory containing the executable
func GetExecutableDir() (string, error) {
	executable, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Dir(executable), nil
}

// GetConfigPath returns the path to a config file relative to the executable
func GetConfigPath(filename string) (string, error) {
	execDir, err := GetExecutableDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(execDir, "configs", filename), nil
}

// GetWebPath returns the path to the web directory relative to the executable
func GetWebPath() (string, error) {
	execDir, err := GetExecutableDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(execDir, "web"), nil
}
