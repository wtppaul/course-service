// internal/config/config.go
package config

import (
	"os"

	"github.com/joho/godotenv"
)

func Load() {
	_ = godotenv.Load()
}

// GetEnv returns env value or default if missing
func GetEnv(key, defaultValue string) string {
	val := os.Getenv(key)
	if val == "" {
		return defaultValue
	}
	return val
}
