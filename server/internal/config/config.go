package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port          int
	DataDir       string
	Aria2Secret   string
	Aria2RPCPort  int
	RcloneRPCPort int
	APIKey        string
}

func Load() *Config {
	return &Config{
		Port:          getEnvInt("PORT", 8080),
		DataDir:       getEnv("DATA_DIR", ".gravity"),
		Aria2Secret:   getEnv("ARIA2_SECRET", "gravity-secret"),
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
