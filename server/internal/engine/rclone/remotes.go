package rclone

import (
	"fmt"
	"strings"

	"gravity/internal/logger"

	_ "github.com/rclone/rclone/backend/combine"
	_ "github.com/rclone/rclone/backend/memory"
	"github.com/rclone/rclone/fs/config"
	"github.com/rclone/rclone/fs/config/configfile"
	"go.uber.org/zap"
)

const GravityRootRemote = "GravityRoot"

// SyncGravityRoot ensures the "GravityRoot" combine remote is configured with all current remotes
func SyncGravityRoot(configPath string) error {
	// Install the config file handler and load from disk
	if configPath != "" {
		config.SetConfigPath(configPath)
	}
	configfile.Install()
	if err := config.Data().Load(); err != nil {
		logger.L.Warn("failed to load rclone config", zap.Error(err), zap.String("path", config.GetConfigPath()))
	}

	remotes := config.GetRemoteNames()
	logger.L.Info("rclone remotes loaded", 
		zap.Int("count", len(remotes)), 
		zap.Strings("remotes", remotes),
		zap.String("config_path", config.GetConfigPath()))

	var upstreams []string
	for _, r := range remotes {
		if r == GravityRootRemote {
			continue
		}
		// Format: RemoteName=RemoteName:
		upstreams = append(upstreams, fmt.Sprintf("%s=%s:", r, r))
	}

	if len(upstreams) == 0 {
		// If no remotes, use 'memory' backend to provide an empty valid root
		// This prevents the engine from crashing on startup due to missing [GravityRoot] section
		config.Data().SetValue(GravityRootRemote, "type", "memory")
		config.Data().DeleteKey(GravityRootRemote, "upstreams")
	} else {
		// Create or Update the remote
		config.Data().SetValue(GravityRootRemote, "type", "combine")
		config.Data().SetValue(GravityRootRemote, "upstreams", strings.Join(upstreams, " "))
	}

	return config.Data().Save()
}
