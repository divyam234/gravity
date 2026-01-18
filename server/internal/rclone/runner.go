package rclone

import (
	"fmt"

	"aria2-rclone-ui/internal/process"
)

type Runner struct {
	proc *process.Process
}

func NewRunner(rcAddr string) *Runner {
	args := []string{
		"rcd",
		fmt.Sprintf("--rc-addr=%s", rcAddr),
		"--rc-no-auth",
		"--rc-allow-origin=*", // Allow CORS
	}

	return &Runner{
		proc: process.New("rclone", args),
	}
}

func (r *Runner) Start() error {
	return r.proc.Start()
}

func (r *Runner) Stop() error {
	return r.proc.Stop()
}
