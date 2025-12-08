package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/joho/godotenv"
)

var (
	once    sync.Once
	loadErr error
)

// Load reads the .env file once and loads variables into the environment.
// It's called automatically by GetConfig, but you can call it explicitly
// at startup if you want to handle errors early.
func Load() error {
	once.Do(func() {
		exePath, _ := os.Executable()
		fmt.Println("exePath:", exePath)

		envPath := filepath.Join(filepath.Dir(exePath), ".env")
		if _, err := os.Stat(envPath); os.IsNotExist(err) {
			log.Fatal(".env file does not exist at:", envPath)
		}

		err := godotenv.Load(envPath)
		if err != nil {
			log.Fatal(".env file failed to load!", err)
		}
	})
	return loadErr
}

// GetConfig retrieves a configuration value by key.
// The .env file is loaded only once on the first call.
func GetConfig(key string) string {
	// Ensure .env is loaded (only happens once)
	Load()
	return os.Getenv(key)
}

// MustGetConfig retrieves a configuration value and panics if not found.
func MustGetConfig(key string) string {
	Load()
	val := os.Getenv(key)
	if val == "" {
		panic(fmt.Sprintf("required config key %q not found", key))
	}
	return val
}

// GetConfigWithDefault retrieves a config value or returns a default.
func GetConfigWithDefault(key, defaultValue string) string {
	Load()
	val := os.Getenv(key)
	if val == "" {
		return defaultValue
	}
	return val
}
