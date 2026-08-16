package tunnel

import (
        "bufio"
        "fmt"
        "io"
        "net/http"
        "os"
        "os/exec"
        "path/filepath"
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

// ── Auto-download ─────────────────────────────────────────────────────────────

// FindOrDownloadCloudflared looks for cloudflared in common locations and,
// if not found, downloads it automatically to dataDir.
func FindOrDownloadCloudflared(dataDir string, onProgress func(string)) (string, error) {
        // 1. Check PATH
        if path, err := exec.LookPath("cloudflared"); err == nil {
                return path, nil
        }

        // 2. Check dataDir copy (from a previous auto-download)
        localBin := localBinaryPath(dataDir)
        if _, err := os.Stat(localBin); err == nil {
                if CheckCloudflared(localBin) {
                        return localBin, nil
                }
        }

        // 3. Auto-download
        if onProgress != nil {
                onProgress("Download cloudflared in corso...")
        }
        if err := downloadCloudflared(localBin); err != nil {
                return "", fmt.Errorf("impossibile scaricare cloudflared: %w", err)
        }
        if onProgress != nil {
                onProgress("cloudflared scaricato")
        }
        return localBin, nil
}

func localBinaryPath(dataDir string) string {
        name := "cloudflared"
        if runtime.GOOS == "windows" {
                name = "cloudflared.exe"
        }
        return dataDir + string(os.PathSeparator) + name
}

// downloadURL returns the GitHub release URL for cloudflared on the current platform.
func downloadURL() string {
        base := "https://github.com/cloudflare/cloudflared/releases/latest/download/"
        switch runtime.GOOS {
        case "windows":
                return base + "cloudflared-windows-amd64.exe"
        case "darwin":
                if runtime.GOARCH == "arm64" {
                        return base + "cloudflared-darwin-arm64"
                }
                return base + "cloudflared-darwin-amd64"
        default:
                if runtime.GOARCH == "arm64" {
                        return base + "cloudflared-linux-arm64"
                }
                return base + "cloudflared-linux-amd64"
        }
}

func downloadCloudflared(destPath string) error {
        if err := os.MkdirAll(filepath.Dir(destPath), 0700); err != nil {
                return err
        }

        resp, err := http.Get(downloadURL())
        if err != nil {
                return err
        }
        defer resp.Body.Close()
        if resp.StatusCode != http.StatusOK {
                return fmt.Errorf("HTTP %d", resp.StatusCode)
        }

        tmp := destPath + ".tmp"
        f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0700)
        if err != nil {
                return err
        }
        if _, err := io.Copy(f, resp.Body); err != nil {
                f.Close()
                os.Remove(tmp)
                return err
        }
        f.Close()
        return os.Rename(tmp, destPath)
}

func CheckCloudflared(bin string) bool {
        out, err := exec.Command(bin, "version").Output()
        if err != nil {
                return false
        }
        return strings.Contains(string(out), "cloudflared")
}
