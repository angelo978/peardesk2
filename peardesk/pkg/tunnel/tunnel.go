package tunnel

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"time"
)

var urlPattern = regexp.MustCompile(`https://[a-zA-Z0-9\-]+\.trycloudflare\.com`)

type Tunnel struct {
	cmd       *exec.Cmd
	URL       string
	LocalPort int
	LogCh     chan string
	ErrCh     chan error
}

func Start(localPort int, cloudflaredBin string) (*Tunnel, error) {
	addr := fmt.Sprintf("http://localhost:%d", localPort)
	cmd := exec.Command(cloudflaredBin, "tunnel", "--url", addr, "--no-autoupdate")
	cmd.Env = append(os.Environ(), "NO_COLOR=1")

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("impossibile avviare cloudflared: %w", err)
	}

	t := &Tunnel{
		cmd:       cmd,
		LocalPort: localPort,
		LogCh:     make(chan string, 100),
		ErrCh:     make(chan error, 1),
	}

	urlCh := make(chan string, 1)

	go func() {
		scanner := bufio.NewScanner(io.MultiReader(stderr, stdout))
		for scanner.Scan() {
			line := scanner.Text()
			t.LogCh <- line
			if match := urlPattern.FindString(line); match != "" {
				select {
				case urlCh <- match:
				default:
				}
			}
		}
		t.ErrCh <- cmd.Wait()
	}()

	select {
	case url := <-urlCh:
		t.URL = url
		return t, nil
	case err := <-t.ErrCh:
		return nil, fmt.Errorf("cloudflared terminato prima di fornire URL: %v", err)
	case <-time.After(30 * time.Second):
		cmd.Process.Kill()
		return nil, fmt.Errorf("timeout: cloudflared non ha fornito un URL entro 30s")
	}
}

func (t *Tunnel) Stop() {
	if t.cmd != nil && t.cmd.Process != nil {
		t.cmd.Process.Kill()
	}
}

func CloudflaredBinaryName() string {
	switch runtime.GOOS {
	case "windows":
		return "cloudflared.exe"
	case "darwin":
		return "cloudflared-darwin"
	default:
		return "cloudflared-linux-amd64"
	}
}

func FindOrDownloadCloudflared(dataDir string) (string, error) {
	candidates := []string{
		"cloudflared",
		"/usr/bin/cloudflared",
		"/usr/local/bin/cloudflared",
		dataDir + "/cloudflared",
		dataDir + "/cloudflared.exe",
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c, nil
		}
		if path, err := exec.LookPath(c); err == nil {
			return path, nil
		}
	}

	// Try system PATH
	if path, err := exec.LookPath("cloudflared"); err == nil {
		return path, nil
	}

	return "", fmt.Errorf(
		"cloudflared non trovato.\n\n" +
			"Installa cloudflared:\n" +
			"  Linux: sudo apt install cloudflared  oppure  brew install cloudflared\n" +
			"  Windows: winget install Cloudflare.cloudflared\n" +
			"  macOS: brew install cloudflare/cloudflare/cloudflared\n\n" +
			"Oppure scaricalo da: https://developers.cloudflare.com/cloudflare-one/connections/connect-apps/install-and-setup/installation/",
	)
}

func CheckCloudflared(bin string) bool {
	out, err := exec.Command(bin, "version").Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(out), "cloudflared")
}
