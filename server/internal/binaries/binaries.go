package binaries

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"gravity/internal/logger"

	"go.uber.org/zap"
)

const (
	Aria2RequiredVersion = "1.37.0"
	YtDlpRequiredVersion = "2026.01.29"
)

type Manager struct {
	binDir string
}

func NewManager(dataDir string) *Manager {
	dir := filepath.Join(dataDir, "bin")
	_ = os.MkdirAll(dir, 0755)
	return &Manager{binDir: dir}
}

// EnsureBinaries checks if required binaries are in PATH or binDir, and downloads if needed
func (m *Manager) EnsureBinaries(ctx context.Context) error {
	// 1. Aria2c
	if err := m.ensureAria2(ctx); err != nil {
		logger.L.Warn("failed to ensure aria2c", zap.Error(err))
	}

	// 2. yt-dlp
	if err := m.ensureYtDlp(ctx); err != nil {
		logger.L.Warn("failed to ensure yt-dlp", zap.Error(err))
	}

	return nil
}

func (m *Manager) GetAria2Path() string {
	// Try local first
	local := filepath.Join(m.binDir, "aria2c")
	if isExecutable(local) {
		return local
	}
	// Fallback to path
	p, err := exec.LookPath("aria2c")
	if err == nil {
		return p
	}
	return "aria2c"
}

func (m *Manager) GetYtDlpPath() string {
	local := filepath.Join(m.binDir, "yt-dlp")
	if isExecutable(local) {
		return local
	}
	p, err := exec.LookPath("yt-dlp")
	if err == nil {
		return p
	}
	return "yt-dlp"
}

func (m *Manager) ensureAria2(ctx context.Context) error {
	path, err := exec.LookPath("aria2c")
	if err == nil {
		if m.checkVersion(path, Aria2RequiredVersion) {
			return nil // System version is fine
		}
	}

	localPath := filepath.Join(m.binDir, "aria2c")
	if isExecutable(localPath) {
		if m.checkVersion(localPath, Aria2RequiredVersion) {
			return nil
		}
	}

	// Need to download
	logger.L.Info("downloading aria2c...", zap.String("version", Aria2RequiredVersion))
	return m.downloadAria2(ctx)
}

func (m *Manager) ensureYtDlp(ctx context.Context) error {
	path, _ := exec.LookPath("yt-dlp")
	if path != "" {
		if m.checkVersion(path, YtDlpRequiredVersion) {
			return nil
		}
	}

	localPath := filepath.Join(m.binDir, "yt-dlp")
	if isExecutable(localPath) {
		if m.checkVersion(localPath, YtDlpRequiredVersion) {
			return nil
		}
	}

	logger.L.Info("downloading yt-dlp...", zap.String("version", YtDlpRequiredVersion))
	return m.downloadYtDlp(ctx)
}

func (m *Manager) checkVersion(path string, required string) bool {
	cmd := exec.Command(path, "--version")
	out, err := cmd.Output()
	if err != nil {
		return false
	}

	if required == "" {
		return true
	}

	return strings.Contains(string(out), required)
}

func (m *Manager) downloadAria2(ctx context.Context) error {
	var arch string
	switch runtime.GOARCH {
	case "amd64":
		arch = "amd64"
	case "arm64":
		arch = "arm64"
	default:
		return fmt.Errorf("unsupported architecture for aria2c download: %s", runtime.GOARCH)
	}

	url := fmt.Sprintf("https://github.com/divyam234/aria2c-static/releases/download/release-%s/aria2-linux-%s.tar.gz", Aria2RequiredVersion, arch)

	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	gr, err := gzip.NewReader(resp.Body)
	if err != nil {
		return err
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if header.Name == "aria2c" {
			f, err := os.OpenFile(filepath.Join(m.binDir, "aria2c"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
			if err != nil {
				return err
			}
			defer f.Close()
			_, err = io.Copy(f, tr)
			return err
		}
	}

	return fmt.Errorf("aria2c not found in tarball")
}

func (m *Manager) downloadYtDlp(ctx context.Context) error {
	suffix := "linux"
	if runtime.GOARCH == "arm64" {
		suffix = "linux_aarch64"
	}

	url := fmt.Sprintf("https://github.com/yt-dlp/yt-dlp/releases/download/%s/yt-dlp_%s", YtDlpRequiredVersion, suffix)

	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	f, err := os.OpenFile(filepath.Join(m.binDir, "yt-dlp"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}

func isExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.Mode()&0111 != 0
}
