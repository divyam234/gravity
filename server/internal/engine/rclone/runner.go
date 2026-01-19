package rclone

import (
	"fmt"
	"log"
	"os/exec"
	"syscall"
	"time"
)

type Runner struct {
	port int
	cmd  *exec.Cmd
}

func NewRunner(port int) *Runner {
	return &Runner{port: port}
}

func (r *Runner) Start() error {
	args := []string{
		"rcd",
		"--rc-no-auth",
		fmt.Sprintf("--rc-addr=localhost:%d", r.port),
		"--rc-allow-origin=*",
		"--rc-serve",
	}

	r.cmd = exec.Command("rclone", args...)
	// Suppress rclone console output
	r.cmd.Stdout = nil
	r.cmd.Stderr = nil

	if err := r.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start rclone rcd: %w", err)
	}

	log.Printf("Rclone rcd started on port %d (PID %d)", r.port, r.cmd.Process.Pid)
	return nil
}

func (r *Runner) Stop() error {
	if r.cmd != nil && r.cmd.Process != nil {
		r.cmd.Process.Signal(syscall.SIGTERM)

		done := make(chan error, 1)
		go func() {
			done <- r.cmd.Wait()
		}()

		select {
		case <-done:
			return nil
		case <-time.After(5 * time.Second):
			r.cmd.Process.Kill()
			return fmt.Errorf("rclone failed to stop gracefully, killed")
		}
	}
	return nil
}
