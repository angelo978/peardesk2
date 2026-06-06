//go:build headless

package host

func injectMouseMove(x, y int)                              {}
func injectMouseClick(x, y int, button string, down bool)   {}
func injectMouseScroll(x, y int, dy float64)                {}
func injectKeyEvent(key string, down bool, modifiers []string) {}
