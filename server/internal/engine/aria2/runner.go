package aria2

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
)

type Runner struct {
	port        int
	secret      string
	sessionFile string
	downloadDir string
	cmd         *exec.Cmd
}

func NewRunner(port int, secret, dataDir string) *Runner {
	os.MkdirAll(dataDir, 0755)
	sessionFile := filepath.Join(dataDir, "aria2.session")
	if _, err := os.Stat(sessionFile); os.IsNotExist(err) {
		os.WriteFile(sessionFile, []byte{}, 0644)
	}

	downloadDir := filepath.Join(dataDir, "downloads")
	os.MkdirAll(downloadDir, 0755)

	return &Runner{
		port:        port,
		secret:      secret,
		sessionFile: sessionFile,
		downloadDir: downloadDir,
	}
}

func (r *Runner) Start() error {
	args := []string{
		"--enable-rpc",
		"--rpc-listen-all=false",
		fmt.Sprintf("--rpc-listen-port=%d", r.port),
		fmt.Sprintf("--rpc-secret=%s", r.secret),
		"--rpc-allow-origin-all=true",
		fmt.Sprintf("--input-file=%s", r.sessionFile),
		fmt.Sprintf("--save-session=%s", r.sessionFile),
		"--save-session-interval=60",
		fmt.Sprintf("--dir=%s", r.downloadDir),
		"--continue=true",
		"--max-connection-per-server=16",
		"--split=16",
		"--min-split-size=1M",
	}

	r.cmd = exec.Command("aria2c", args...)
	// Suppress aria2c console output
	r.cmd.Stdout = os.Stdout
	r.cmd.Stderr = os.Stderr

	if err := r.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start aria2c: %w", err)
	}

	log.Printf("Aria2 started on port %d (PID %d)", r.port, r.cmd.Process.Pid)
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
			return fmt.Errorf("aria2c failed to stop gracefully, killed")
		}
	}
	return nil
}

func (r *Runner) DownloadDir() string {
	return r.downloadDir
}
