package core

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

// LoadSecrets loads secrets from a .env file securely
func LoadSecrets(path string) (map[string]string, error) {
	// check file permissions
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // treat missing secrets file as empty map
		}
		return nil, err
	}

	// Warn if permissions are too open (more permissive than 0600)
	// On Unix, 0600 is -rw-------
	mode := info.Mode().Perm()
	if mode&0077 != 0 {
		fmt.Fprintf(os.Stderr, "WARNING: Secrets file %s has insecure permissions (%o). It should be 0600.\n", path, mode)
	}

	return godotenv.Read(path)
}
