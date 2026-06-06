//go:build !headless

package host

import (
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
	if dy > 0 {
		robotgo.Scroll(0, -int(dy))
	} else {
		robotgo.Scroll(0, -int(dy))
	}
}

func injectKeyEvent(key string, down bool, modifiers []string) {
	_ = modifiers
	if down {
		robotgo.KeyDown(key)
	} else {
		robotgo.KeyUp(key)
	}
}
