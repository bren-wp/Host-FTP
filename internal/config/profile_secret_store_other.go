//go:build !darwin

package config

import "github.com/bren-wp/Host-FTP/internal/security"

func protectStoredProfileSecret(value string) (string, error) {
	return security.ProtectString(value)
}
