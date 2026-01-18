package aria2

import (
	"fmt"
	"os"
	"path/filepath"

	"aria2-rclone-ui/internal/process"
)

type Runner struct {
	proc *process.Process
}

func NewRunner(rpcPort int, rpcSecret string) *Runner {
	args := []string{
		"--enable-rpc",
		"--rpc-listen-all=false",
		"--rpc-allow-origin-all",
		fmt.Sprintf("--rpc-listen-port=%d", rpcPort),
		fmt.Sprintf("--rpc-secret=%s", rpcSecret),
		"--continue=true",
		// "--rpc-secure=false", // Default
	}

	// Add default download dir if needed, or let user config handle it
	home, _ := os.UserHomeDir()
	dlDir := filepath.Join(home, "Downloads")
	args = append(args, fmt.Sprintf("--dir=%s", dlDir))

	return &Runner{
		proc: process.New("aria2c", args),
	}
}

func (r *Runner) Start() error {
	return r.proc.Start()
}

func (r *Runner) Stop() error {
	return r.proc.Stop()
}
