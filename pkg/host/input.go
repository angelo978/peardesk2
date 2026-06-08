//go:build !headless

package host

import (
	"strings"

	"github.com/go-vgo/robotgo"
)

func injectMouseMove(x, y int) {
	robotgo.Move(x, y)
}

func injectMouseClick(x, y int, button string, down bool) {
	robotgo.Move(x, y)
	var btn string
	switch button {
	case "right":
		btn = "right"
	case "middle":
		btn = "center"
	default:
		btn = "left"
	}
	if down {
		robotgo.MouseDown(btn)
	} else {
		robotgo.MouseUp(btn)
	}
}

func injectMouseScroll(x, y int, dy float64) {
	robotgo.Move(x, y)
	ticks := int(dy)
	if ticks == 0 {
		if dy > 0 {
			ticks = 1
		} else {
			ticks = -1
		}
	}
	robotgo.Scroll(0, -ticks)
}

// injectRune types a single Unicode character as-is (handles uppercase, @, etc.)
func injectRune(ch string) {
	robotgo.TypeStr(ch)
}

func injectKeyEvent(key string, down bool, modifiers []string) {
	mapped := mapKey(key)
	if mapped == "" {
		return
	}
	// Inject modifiers first on keydown
	if down {
		for _, mod := range modifiers {
			m := mapModifier(mod)
			if m != "" {
				robotgo.KeyDown(m)
			}
		}
		robotgo.KeyDown(mapped)
	} else {
		robotgo.KeyUp(mapped)
		for i := len(modifiers) - 1; i >= 0; i-- {
			m := mapModifier(modifiers[i])
			if m != "" {
				robotgo.KeyUp(m)
			}
		}
	}
}

// mapModifier converts a Fyne modifier name to robotgo name.
func mapModifier(mod string) string {
	switch strings.ToLower(mod) {
	case "shift", "leftshift", "rightshift":
		return "shift"
	case "control", "ctrl", "leftcontrol", "rightcontrol":
		return "ctrl"
	case "alt", "leftalt", "rightalt":
		return "alt"
	case "super", "leftsuper", "rightsuper", "meta", "win":
		return "cmd"
	}
	return ""
}

// mapKey converts a Fyne KeyName to a robotgo key string.
func mapKey(key string) string {
	switch key {
	// Whitespace / control
	case "Return", "KP_Enter":
		return "enter"
	case "BackSpace":
		return "backspace"
	case "Delete":
		return "delete"
	case "Escape":
		return "escape"
	case "Tab":
		return "tab"
	case "Space":
		return "space"
	case "Insert":
		return "insert"

	// Navigation
	case "Up":
		return "up"
	case "Down":
		return "down"
	case "Left":
		return "left"
	case "Right":
		return "right"
	case "Home":
		return "home"
	case "End":
		return "end"
	case "PageUp":
		return "pageup"
	case "PageDown":
		return "pagedown"

	// Function keys
	case "F1":
		return "f1"
	case "F2":
		return "f2"
	case "F3":
		return "f3"
	case "F4":
		return "f4"
	case "F5":
		return "f5"
	case "F6":
		return "f6"
	case "F7":
		return "f7"
	case "F8":
		return "f8"
	case "F9":
		return "f9"
	case "F10":
		return "f10"
	case "F11":
		return "f11"
	case "F12":
		return "f12"

	// Modifiers (standalone)
	case "LeftShift", "RightShift":
		return "shift"
	case "LeftControl", "RightControl":
		return "ctrl"
	case "LeftAlt", "RightAlt":
		return "alt"
	case "LeftSuper", "RightSuper":
		return "cmd"
	case "CapsLock":
		return "capslock"
	case "NumLock":
		return "numlock"
	case "PrintScreen":
		return "printscreen"
	case "ScrollLock":
		return "scrolllock"
	}

	// Single character keys: a-z, 0-9, punctuation
	if len(key) == 1 {
		return strings.ToLower(key)
	}
	return ""
}
