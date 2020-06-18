package config

import (
	"os"
	"path/filepath"
)

var (
	// CAFile defines the path to the CA certificate
	CAFile = configFile("ca.pem")

	// ServerCertFile defines the path to the Server certificate
	ServerCertFile = configFile("server.pem")
	// ServerKeyFile defines the path to the Server private key
	ServerKeyFile = configFile("server-key.pem")

	// RootClientCertFile defines the path to the Client certificate
	// with 'root' level access
	RootClientCertFile = configFile("root-client.pem")
	// RootClientKeyFile defines the path to the Client private key
	// with 'root' level access
	RootClientKeyFile = configFile("root-client-key.pem")

	// NobodyClientCertFile defines the path to the Client certificate
	// with 'nobody' level access
	NobodyClientCertFile = configFile("nobody-client.pem")
	// NobodyClientKeyFile defines the path to the Client private key
	// with 'nobody' level access
	NobodyClientKeyFile = configFile("nobody-client-key.pem")
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
