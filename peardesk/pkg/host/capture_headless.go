//go:build headless

package host

func captureScreen(maxWidth, maxHeight int) (string, int, int, error) {
	return "", 0, 0, nil
}

func screenSize() (int, int) {
	return 1920, 1080
}
