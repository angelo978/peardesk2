//go:build darwin

package autostart

import (
	"os"
	"path/filepath"
	"strings"
)

func launchAgentDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "LaunchAgents")
}

func plistPath() string {
	return filepath.Join(launchAgentDir(), "com.peardesk.app.plist")
}

func isEnabled() bool {
	_, err := os.Stat(plistPath())
	return err == nil
}

func enable(execPath string) error {
	if err := os.MkdirAll(launchAgentDir(), 0755); err != nil {
		return err
	}
	const tmpl = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
  "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>             <string>com.peardesk.app</string>
  <key>ProgramArguments</key>
  <array><string>EXEC_PATH</string></array>
  <key>RunAtLoad</key>         <true/>
  <key>KeepAlive</key>         <false/>
</dict>
</plist>
`
	content := strings.ReplaceAll(tmpl, "EXEC_PATH", execPath)
	return os.WriteFile(plistPath(), []byte(content), 0644)
}

func disable() error {
	if err := os.Remove(plistPath()); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
