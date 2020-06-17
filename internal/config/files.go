package config

import (
	"os"
	"path/filepath"
)

var (
	// CAFile defines the path to CA certificate
	CAFile = configFile("ca.pem")
	// ServerCertFile defines the path to Server certificate
	ServerCertFile = configFile("server.pem")
	// ServerKeyFile defines the path to Server private key
	ServerKeyFile = configFile("server-key.pem")
)

const (
	proglog = ".proglog"
)

func configFile(filename string) string {
	if dir := os.Getenv("CONFIG_DIR"); dir != "" {
		return filepath.Join(dir, filename)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	return filepath.Join(homeDir, proglog, filename)
}
