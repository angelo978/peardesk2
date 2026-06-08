//go:build windows

package autostart

import (
	"golang.org/x/sys/windows/registry"
)

const regKey = `Software\Microsoft\Windows\CurrentVersion\Run`
const regValue = "PearDesk"

func isEnabled() bool {
	k, err := registry.OpenKey(registry.CURRENT_USER, regKey, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()
	_, _, err = k.GetStringValue(regValue)
	return err == nil
}

func enable(execPath string) error {
	k, _, err := registry.CreateKey(registry.CURRENT_USER, regKey, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	return k.SetStringValue(regValue, execPath)
}

func disable() error {
	k, err := registry.OpenKey(registry.CURRENT_USER, regKey, registry.SET_VALUE)
	if err != nil {
		return nil
	}
	defer k.Close()
	err = k.DeleteValue(regValue)
	if err != nil && err.Error() == "The system cannot find the file specified." {
		return nil
	}
	return err
}
