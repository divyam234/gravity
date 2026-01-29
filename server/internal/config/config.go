package config

import (
	"os"
	"path/filepath"
	"strings"

	"gravity/internal/logger"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	"go.uber.org/zap"
)

type Config struct {
	Port         int    `koanf:"port"`
	LogLevel     string `koanf:"log_level"`
	DataDir      string `koanf:"data_dir"`
	Aria2RPCPort int    `koanf:"aria2_rpc_port"`

	RcloneConfigPath string   `koanf:"rclone_config_path"`
	APIKey           string   `koanf:"api_key"`
	Database         DBConfig `koanf:"database"`

	LogFile string `koanf:"log_file"`
	JSONLog bool   `koanf:"json_log"`
}

type DBConfig struct {
	Type string `koanf:"type"` // sqlite or postgres
	DSN  string `koanf:"dsn"`  // Data Source Name
}

func Load() *Config {
	k := koanf.New(".")

	// 1. Set defaults
	home, _ := os.UserHomeDir()
	defaultDataDir := filepath.Join(home, ".gravity")

	k.Set("port", 8080)
	k.Set("log_level", "info")
	k.Set("data_dir", defaultDataDir)
	k.Set("aria2_rpc_port", 6800)
	k.Set("rclone_rpc_port", 5572)
	k.Set("database.type", "sqlite")
	k.Set("database.dsn", filepath.Join(defaultDataDir, "gravity.db"))

	// 2. Load from config file (optional)
	if err := k.Load(file.Provider("config.yaml"), yaml.Parser()); err != nil {
		logger.L.Warn("no config.yaml found, using defaults and environment", zap.Error(err))
	}

	// 3. Load from environment variables (GRAVITY_PORT, etc.)
	k.Load(env.Provider("GRAVITY_", ".", func(s string) string {
		key := strings.TrimPrefix(s, "GRAVITY_")
		key = strings.ToLower(key)
		return strings.ReplaceAll(key, "__", ".")
	}), nil)

	var cfg Config
	if err := k.Unmarshal("", &cfg); err != nil {
		logger.L.Fatal("error unmarshalling config", zap.Error(err))
	}

	// 4. Post-processing: If DSN is default and DataDir changed, update DSN
	if cfg.Database.Type == "sqlite" && (cfg.Database.DSN == "" || strings.Contains(cfg.Database.DSN, ".gravity/gravity.db")) {
		os.MkdirAll(cfg.DataDir, 0755)
		cfg.Database.DSN = filepath.Join(cfg.DataDir, "gravity.db")
	}

	return &cfg
}
