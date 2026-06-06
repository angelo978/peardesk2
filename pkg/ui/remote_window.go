package ui

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/jpeg"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/peardesk/peardesk/pkg/clipboard"
	"github.com/peardesk/peardesk/pkg/client"
)

type RemoteWindow struct {
	win       fyne.Window
	conn      *client.Connection
	imgWidget *canvas.Image
	statusLbl *widget.Label
	cbLbl     *widget.Label // clipboard sync indicator
	mu        sync.Mutex
	remoteW   int
	remoteH   int
	app       fyne.App
	hostID    string
	cbMon     *clipboard.Monitor
}

func ShowRemoteWindow(a fyne.App, conn *client.Connection, hostID string) *RemoteWindow {
	win := a.NewWindow("PearDesk — " + hostID)
	win.Resize(fyne.NewSize(1280, 720))

	rw := &RemoteWindow{
		app:       a,
		win:       win,
		conn:      conn,
		statusLbl: widget.NewLabel("Connesso a " + hostID),
		cbLbl:     widget.NewLabel(""),
		hostID:    hostID,
	}

	img := canvas.NewImageFromImage(image.NewRGBA(image.Rect(0, 0, 1280, 720)))
	img.FillMode = canvas.ImageFillContain
	img.ScaleMode = canvas.ImageScaleFastest
	rw.imgWidget = img

	interactiveImg := newInteractiveImage(img, rw)

	filesBtn := widget.NewButtonWithIcon("File", theme.FolderOpenIcon(), func() {
		ShowFileWindow(a, conn, hostID)
	})

	toolbar := container.NewBorder(nil, nil,
		rw.statusLbl,
		container.NewHBox(rw.cbLbl, filesBtn),
		nil,
	)
	content := container.NewBorder(toolbar, nil, nil, nil, interactiveImg)
	win.SetContent(content)

	win.Canvas().SetOnTypedKey(func(ev *fyne.KeyEvent) {
		key := string(ev.Name)
		rw.conn.SendKeyDown(key, nil)
		rw.conn.SendKeyUp(key, nil)
	})

	// Video stream callbacks
	conn.OnFrame = func(b64 string, w, h int) { rw.updateFrame(b64, w, h) }
	conn.OnClose = func() { rw.statusLbl.SetText("Connessione chiusa") }
	conn.OnError = func(err error) { rw.statusLbl.SetText("Errore: " + err.Error()) }

	// Clipboard: host → client
	conn.OnClipboard = func(text string) {
		if err := rw.cbMon.Write(text); err == nil {
			rw.flashClipboard("← host")
		}
	}

	// Clipboard: client → host (monitor local clipboard)
	rw.cbMon = clipboard.New(500 * time.Millisecond)
	rw.cbMon.Start(func(text string) {
		conn.SendClipboard(text)
		rw.flashClipboard("→ host")
	})

	win.SetOnClosed(func() {
		rw.cbMon.Stop()
		conn.Close()
	})
	win.Show()
	return rw
}

// flashClipboard shows a brief "📋 direction" label that fades after 2 s.
func (rw *RemoteWindow) flashClipboard(direction string) {
	rw.cbLbl.SetText(fmt.Sprintf("📋 %s", direction))
	go func() {
		time.Sleep(2 * time.Second)
		rw.cbLbl.SetText("")
	}()
}

func (rw *RemoteWindow) updateFrame(b64 string, w, h int) {
	data, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return
	}
	img, err := jpeg.Decode(bytes.NewReader(data))
	if err != nil {
		return
	}
	rw.mu.Lock()
	rw.remoteW = w
	rw.remoteH = h
	rw.mu.Unlock()
	rw.imgWidget.Image = img
	rw.imgWidget.Refresh()
}

// ─── Interactive image widget ─────────────────────────────────────────────────

type interactiveImage struct {
	widget.BaseWidget
	img *canvas.Image
	rw  *RemoteWindow
}

func newInteractiveImage(img *canvas.Image, rw *RemoteWindow) *interactiveImage {
	i := &interactiveImage{img: img, rw: rw}
	i.ExtendBaseWidget(i)
	return i
}

func (i *interactiveImage) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(i.img)
}

func (i *interactiveImage) MouseIn(_ *desktop.MouseEvent)  {}
func (i *interactiveImage) MouseOut()                       {}
func (i *interactiveImage) MouseMoved(ev *desktop.MouseEvent) {
	xR, yR := i.ratios(ev.Position)
	i.rw.conn.SendMouseMove(xR, yR)
}

func (i *interactiveImage) Tapped(ev *fyne.PointEvent) {
	xR, yR := i.ratios(ev.Position)
	i.rw.conn.SendMouseDown(xR, yR, "left")
	i.rw.conn.SendMouseUp(xR, yR, "left")
}

func (i *interactiveImage) TappedSecondary(ev *fyne.PointEvent) {
	xR, yR := i.ratios(ev.Position)
	i.rw.conn.SendMouseDown(xR, yR, "right")
	i.rw.conn.SendMouseUp(xR, yR, "right")
}

func (i *interactiveImage) Scrolled(ev *fyne.ScrollEvent) {
	xR, yR := i.ratios(ev.Position)
	i.rw.conn.SendScroll(xR, yR, float64(ev.Scrolled.DY))
}

func (i *interactiveImage) ratios(pos fyne.Position) (float64, float64) {
	size := i.Size()
	if size.Width == 0 || size.Height == 0 {
		return 0, 0
	}
	return float64(pos.X) / float64(size.Width), float64(pos.Y) / float64(size.Height)
}
