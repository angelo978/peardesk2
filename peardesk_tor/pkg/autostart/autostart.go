// Package autostart manages OS-level auto-start entries.
package autostart

// IsEnabled returns true if PearDesk is registered to start with the OS.
func IsEnabled() bool { return isEnabled() }

// Enable registers PearDesk to start with the OS using execPath as the binary.
func Enable(execPath string) error { return enable(execPath) }

// Disable removes the auto-start entry.
func Disable() error { return disable() }
