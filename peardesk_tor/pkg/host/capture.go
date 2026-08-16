//go:build !headless

package host

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/jpeg"

	"github.com/kbinani/screenshot"
	"github.com/nfnt/resize"
)

func captureScreen(maxWidth, maxHeight int) (string, int, int, error) {
	n := screenshot.NumActiveDisplays()
	if n == 0 {
		return "", 0, 0, nil
	}
	img, err := screenshot.CaptureDisplay(0)
	if err != nil {
		return "", 0, 0, err
	}

	origW := img.Bounds().Dx()
	origH := img.Bounds().Dy()

	var scaled image.Image = img
	if maxWidth > 0 && maxHeight > 0 {
		scale := 1.0
		if float64(origW) > float64(maxWidth) {
			scale = float64(maxWidth) / float64(origW)
		}
		if float64(origH)*scale > float64(maxHeight) {
			scale = float64(maxHeight) / float64(origH)
		}
		if scale < 1.0 {
			newW := uint(float64(origW) * scale)
			newH := uint(float64(origH) * scale)
			scaled = resize.Resize(newW, newH, img, resize.Bilinear)
		}
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, scaled, &jpeg.Options{Quality: 70}); err != nil {
		return "", 0, 0, err
	}

	b64 := base64.StdEncoding.EncodeToString(buf.Bytes())
	return b64, scaled.Bounds().Dx(), scaled.Bounds().Dy(), nil
}

func screenSize() (int, int) {
	n := screenshot.NumActiveDisplays()
	if n == 0 {
		return 1920, 1080
	}
	bounds := screenshot.GetDisplayBounds(0)
	return bounds.Dx(), bounds.Dy()
}
