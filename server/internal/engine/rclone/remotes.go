package rclone

import (
	"fmt"
	"strings"

	_ "github.com/rclone/rclone/backend/combine"
	"github.com/rclone/rclone/fs/config"
)

const GravityRootRemote = "GravityRoot"

// SyncGravityRoot ensures the "GravityRoot" combine remote is configured with all current remotes
func SyncGravityRoot() error {
	remotes := config.GetRemoteNames()

	var upstreams []string
	for _, r := range remotes {
		if r == GravityRootRemote {
			continue
		}
		// Format: RemoteName=RemoteName:
		upstreams = append(upstreams, fmt.Sprintf("%s=%s:", r, r))
	}

	if len(upstreams) == 0 {
		// If no remotes, delete GravityRoot if it exists
		if config.Data().HasSection(GravityRootRemote) {
			config.Data().DeleteSection(GravityRootRemote)
			return config.Data().Save()
		}
		return nil
	}

	// Create or Update the remote
	config.Data().SetValue(GravityRootRemote, "type", "combine")
	config.Data().SetValue(GravityRootRemote, "upstreams", strings.Join(upstreams, " "))

	return config.Data().Save()
}
