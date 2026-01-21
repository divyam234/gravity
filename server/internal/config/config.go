package config

import (
	"os"
	"path/filepath"
	"strconv"
)

type Config struct {
	Port          int
	DataDir       string
	Aria2RPCPort  int
	RcloneRPCPort int
	APIKey        string
}

func Load() *Config {
	home, _ := os.UserHomeDir()
	defaultDataDir := filepath.Join(home, ".gravity")

	return &Config{
		Port:          getEnvInt("PORT", 8080),
		DataDir:       getEnv("DATA_DIR", defaultDataDir),
		Aria2RPCPort:  getEnvInt("ARIA2_RPC_PORT", 6800),
		RcloneRPCPort: getEnvInt("RCLONE_RPC_PORT", 5572),
		APIKey:        getEnv("API_KEY", ""),
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	valueStr := getEnv(key, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return fallback
}
