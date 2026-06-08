//go:build linux

package autostart

import (
	"os"
	"path/filepath"
)

func autostartDir() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "autostart")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "autostart")
}

func desktopPath() string {
	return filepath.Join(autostartDir(), "peardesk.desktop")
}

func isEnabled() bool {
	_, err := os.Stat(desktopPath())
	return err == nil
}

func enable(execPath string) error {
	if err := os.MkdirAll(autostartDir(), 0755); err != nil {
		return err
	}
	content := "[Desktop Entry]\nType=Application\nName=PearDesk\nExec=" + execPath +
		"\nHidden=false\nNoDisplay=false\nX-GNOME-Autostart-enabled=true\n"
	return os.WriteFile(desktopPath(), []byte(content), 0644)
}

func disable() error {
	if err := os.Remove(desktopPath()); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
