package aria2

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"go.uber.org/zap"
)

type Runner struct {
	port        int
	sessionFile string
	downloadDir string
	binaryPath  string
	cmd         *exec.Cmd
	Verbose     bool
	logger      *zap.Logger
}

func NewRunner(port int, dataDir string, l *zap.Logger) *Runner {
	os.MkdirAll(dataDir, 0755)
	sessionFile := filepath.Join(dataDir, "aria2.session")
	if _, err := os.Stat(sessionFile); os.IsNotExist(err) {
		os.WriteFile(sessionFile, []byte{}, 0644)
	}

	downloadDir := filepath.Join(dataDir, "downloads")
	os.MkdirAll(downloadDir, 0755)

	return &Runner{
		port:        port,
		sessionFile: sessionFile,
		downloadDir: downloadDir,
		binaryPath:  "aria2c", // Default
		logger:      l,
	}
}

func (r *Runner) SetBinaryPath(path string) {
	r.binaryPath = path
}

func (r *Runner) Start() error {
	args := []string{
		"--enable-rpc",
		"--rpc-listen-all=false",
		fmt.Sprintf("--rpc-listen-port=%d", r.port),
		"--rpc-allow-origin-all=true",
		fmt.Sprintf("--input-file=%s", r.sessionFile),
		fmt.Sprintf("--save-session=%s", r.sessionFile),
		"--save-session-interval=60",
		"--dir=" + r.downloadDir,
		"--continue=true",
		"--max-connection-per-server=16",
		"--split=16",
		"--min-split-size=1M",
		"--disable-ipv6=true",
	}

	r.cmd = exec.Command(r.binaryPath, args...)

	if r.Verbose {
		r.cmd.Stdout = os.Stdout
		r.cmd.Stderr = os.Stderr
	} else {
		r.cmd.Stdout = nil
		r.cmd.Stderr = nil
	}

	if err := r.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start aria2c: %w", err)
	}

	r.logger.Info("aria2 started", zap.Int("port", r.port), zap.Int("pid", r.cmd.Process.Pid))
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
