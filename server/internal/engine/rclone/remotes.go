package rclone

import (
	"fmt"
	"log"
	"strings"

	_ "github.com/rclone/rclone/backend/combine"
	_ "github.com/rclone/rclone/backend/memory"
	"github.com/rclone/rclone/fs/config"
	"github.com/rclone/rclone/fs/config/configfile"
)

const GravityRootRemote = "GravityRoot"

// SyncGravityRoot ensures the "GravityRoot" combine remote is configured with all current remotes
func SyncGravityRoot() error {
	// Install the config file handler and load from disk
	configfile.Install()
	if err := config.Data().Load(); err != nil {
		log.Printf("Warning: Failed to load rclone config: %v", err)
	}

	remotes := config.GetRemoteNames()
	log.Printf("Rclone: Found %d remotes in config: %v", len(remotes), remotes)

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
