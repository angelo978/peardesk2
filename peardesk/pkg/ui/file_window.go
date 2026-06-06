package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/peardesk/peardesk/pkg/client"
	"github.com/peardesk/peardesk/pkg/protocol"
)

// transferEntry tracks a single active or completed transfer shown in the log.
type transferEntry struct {
	id       string
	name     string
	dir      string // "↑ Upload" or "↓ Download"
	chunks   int64
	total    int64
	done     bool
	errStr   string
	ts       time.Time
}

// FileWindow is the file transfer window.
type FileWindow struct {
	win        fyne.Window
	app        fyne.App
	conn       *client.Connection
	hostID     string

	// Local panel state
	localPath  string
	localFiles []protocol.FileInfo
	localSel   int // selected index (-1 = none)
	localList  *widget.List

	// Remote panel state
	remotePath  string
	remoteFiles []protocol.FileInfo
	remoteSel   int
	remoteList  *widget.List

	// Transfer log
	transfers   []transferEntry
	transfersMu sync.Mutex
	logList     *widget.List

	// Progress bar for the active transfer
	progressBar  *widget.ProgressBar
	progressLbl  *widget.Label
	sendBtn      *widget.Button
	recvBtn      *widget.Button

	// Path labels
	localPathLbl  *widget.Label
	remotePathLbl *widget.Label
}

func ShowFileWindow(app fyne.App, conn *client.Connection, hostID string) *FileWindow {
	fw := &FileWindow{
		app:      app,
		conn:     conn,
		hostID:   hostID,
		localSel: -1,
		remoteSel: -1,
	}

	// Get initial local path
	if home, err := os.UserHomeDir(); err == nil {
		fw.localPath = home
	} else {
		fw.localPath = "/"
	}
	fw.loadLocalFiles()

	// Register file-list callback
	conn.OnFileList = func(path string, files []protocol.FileInfo) {
		fw.remotePath = path
		fw.remoteFiles = files
		fw.remoteSel = -1
		if fw.remotePathLbl != nil {
			fw.remotePathLbl.SetText(shortPath(path))
		}
		if fw.remoteList != nil {
			fw.remoteList.Refresh()
		}
		fw.updateButtons()
	}

	fw.win = app.NewWindow("Trasferimento File — " + hostID)
	fw.win.Resize(fyne.NewSize(1000, 640))
	fw.win.SetContent(fw.buildContent())
	fw.win.SetOnClosed(func() {
		conn.OnFileList = nil
	})
	fw.win.Show()

	// Load remote root
	go conn.RequestFileList("")
	return fw
}

func (fw *FileWindow) buildContent() fyne.CanvasObject {
	// ── Local panel ──────────────────────────────────────────────────────────
	fw.localPathLbl = widget.NewLabel(shortPath(fw.localPath))
	fw.localPathLbl.TextStyle = fyne.TextStyle{Monospace: true}

	localUpBtn := widget.NewButtonWithIcon("", theme.NavigateBackIcon(), fw.localGoUp)
	localRefreshBtn := widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {
		fw.loadLocalFiles()
		fw.localList.Refresh()
		fw.updateButtons()
	})

	fw.localList = widget.NewList(
		func() int { return len(fw.localFiles) },
		func() fyne.CanvasObject {
			icon := widget.NewIcon(theme.FileIcon())
			lbl := widget.NewLabel("")
			lbl.Truncation = fyne.TextTruncateEllipsis
			szLbl := widget.NewLabel("")
			szLbl.TextStyle = fyne.TextStyle{Italic: true}
			szLbl.Alignment = fyne.TextAlignTrailing
			return container.NewBorder(nil, nil, icon,
				container.NewGridWrap(fyne.NewSize(80, 20), szLbl), lbl)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id >= len(fw.localFiles) {
				return
			}
			f := fw.localFiles[id]
			row := obj.(*fyne.Container)
			icon := row.Objects[0].(*widget.Icon)
			lbl := row.Objects[1].(*widget.Label)
			szCont := row.Objects[2].(*fyne.Container)
			szLbl := szCont.Objects[0].(*widget.Label)

			if f.IsDir {
				icon.SetResource(theme.FolderIcon())
				szLbl.SetText("")
			} else {
				icon.SetResource(theme.FileIcon())
				szLbl.SetText(humanSize(f.Size))
			}
			lbl.SetText(f.Name)
		},
	)
	fw.localList.OnSelected = func(id widget.ListItemID) {
		if id >= len(fw.localFiles) {
			return
		}
		f := fw.localFiles[id]
		if f.IsDir {
			fw.localPath = filepath.Join(fw.localPath, f.Name)
			fw.localPathLbl.SetText(shortPath(fw.localPath))
			fw.loadLocalFiles()
			fw.localList.Refresh()
			fw.localSel = -1
		} else {
			fw.localSel = id
		}
		fw.updateButtons()
	}
	fw.localList.OnUnselected = func(_ widget.ListItemID) {
		fw.localSel = -1
		fw.updateButtons()
	}

	localToolbar := container.NewBorder(nil, nil,
		container.NewHBox(localUpBtn, localRefreshBtn),
		nil,
		fw.localPathLbl,
	)
	localPanel := container.NewBorder(
		container.NewVBox(widget.NewLabelWithStyle("Questo PC", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}), localToolbar),
		nil, nil, nil,
		fw.localList,
	)

	// ── Middle buttons ────────────────────────────────────────────────────────
	fw.sendBtn = widget.NewButtonWithIcon("Invia →", theme.MailSendIcon(), fw.doUpload)
	fw.sendBtn.Importance = widget.HighImportance
	fw.sendBtn.Disable()

	fw.recvBtn = widget.NewButtonWithIcon("← Ricevi", theme.DownloadIcon(), fw.doDownload)
	fw.recvBtn.Importance = widget.MediumImportance
	fw.recvBtn.Disable()

	middleBar := container.NewVBox(
		widget.NewLabel(""),
		fw.sendBtn,
		fw.recvBtn,
	)

	// ── Remote panel ──────────────────────────────────────────────────────────
	fw.remotePathLbl = widget.NewLabel("caricamento...")
	fw.remotePathLbl.TextStyle = fyne.TextStyle{Monospace: true}

	remoteUpBtn := widget.NewButtonWithIcon("", theme.NavigateBackIcon(), fw.remoteGoUp)
	remoteRefreshBtn := widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {
		go fw.conn.RequestFileList(fw.remotePath)
	})

	fw.remoteList = widget.NewList(
		func() int { return len(fw.remoteFiles) },
		func() fyne.CanvasObject {
			icon := widget.NewIcon(theme.FileIcon())
			lbl := widget.NewLabel("")
			lbl.Truncation = fyne.TextTruncateEllipsis
			szLbl := widget.NewLabel("")
			szLbl.TextStyle = fyne.TextStyle{Italic: true}
			szLbl.Alignment = fyne.TextAlignTrailing
			return container.NewBorder(nil, nil, icon,
				container.NewGridWrap(fyne.NewSize(80, 20), szLbl), lbl)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id >= len(fw.remoteFiles) {
				return
			}
			f := fw.remoteFiles[id]
			row := obj.(*fyne.Container)
			icon := row.Objects[0].(*widget.Icon)
			lbl := row.Objects[1].(*widget.Label)
			szCont := row.Objects[2].(*fyne.Container)
			szLbl := szCont.Objects[0].(*widget.Label)

			if f.IsDir {
				icon.SetResource(theme.FolderOpenIcon())
				szLbl.SetText("")
			} else {
				icon.SetResource(theme.FileIcon())
				szLbl.SetText(humanSize(f.Size))
			}
			lbl.SetText(f.Name)
		},
	)
	fw.remoteList.OnSelected = func(id widget.ListItemID) {
		if id >= len(fw.remoteFiles) {
			return
		}
		f := fw.remoteFiles[id]
		if f.IsDir {
			newPath := fw.remotePath + "/" + f.Name
			fw.remoteSel = -1
			go fw.conn.RequestFileList(newPath)
		} else {
			fw.remoteSel = id
		}
		fw.updateButtons()
	}
	fw.remoteList.OnUnselected = func(_ widget.ListItemID) {
		fw.remoteSel = -1
		fw.updateButtons()
	}

	remoteToolbar := container.NewBorder(nil, nil,
		container.NewHBox(remoteUpBtn, remoteRefreshBtn),
		nil,
		fw.remotePathLbl,
	)
	remotePanel := container.NewBorder(
		container.NewVBox(widget.NewLabelWithStyle("Host remoto", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}), remoteToolbar),
		nil, nil, nil,
		fw.remoteList,
	)

	// ── File panels split ─────────────────────────────────────────────────────
	fileSplit := container.NewBorder(nil, nil,
		nil, nil,
		container.NewGridWithColumns(3, localPanel, middleBar, remotePanel),
	)

	// ── Progress bar ─────────────────────────────────────────────────────────
	fw.progressBar = widget.NewProgressBar()
	fw.progressBar.Hide()
	fw.progressLbl = widget.NewLabel("")

	progressRow := container.NewVBox(
		container.NewBorder(nil, nil, fw.progressLbl, nil, fw.progressBar),
	)

	// ── Transfer log ──────────────────────────────────────────────────────────
	fw.logList = widget.NewList(
		func() int {
			fw.transfersMu.Lock()
			defer fw.transfersMu.Unlock()
			return len(fw.transfers)
		},
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewLabel(""),  // direction
				widget.NewLabel(""),  // name
				widget.NewLabel(""),  // status
				widget.NewLabel(""),  // time
			)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			fw.transfersMu.Lock()
			if id >= len(fw.transfers) {
				fw.transfersMu.Unlock()
				return
			}
			t := fw.transfers[len(fw.transfers)-1-id] // newest first
			fw.transfersMu.Unlock()

			row := obj.(*fyne.Container)
			row.Objects[0].(*widget.Label).SetText(t.dir)
			row.Objects[1].(*widget.Label).SetText(t.name)
			if t.errStr != "" {
				row.Objects[2].(*widget.Label).SetText("✗ " + t.errStr)
			} else if t.done {
				row.Objects[2].(*widget.Label).SetText(fmt.Sprintf("✓ %d/%d chunk", t.chunks, t.total))
			} else {
				row.Objects[2].(*widget.Label).SetText(fmt.Sprintf("… %d/%d", t.chunks, t.total))
			}
			row.Objects[3].(*widget.Label).SetText(t.ts.Format("15:04:05"))
		},
	)

	logCard := widget.NewCard("Log trasferimenti", "", fw.logList)

	return container.NewBorder(
		nil,
		container.NewVBox(progressRow, logCard),
		nil, nil,
		fileSplit,
	)
}

// ─── Local navigation ─────────────────────────────────────────────────────────

func (fw *FileWindow) loadLocalFiles() {
	entries, err := os.ReadDir(fw.localPath)
	if err != nil {
		fw.localFiles = nil
		return
	}
	files := make([]protocol.FileInfo, 0, len(entries))
	for _, e := range entries {
		info, _ := e.Info()
		var size int64
		if info != nil {
			size = info.Size()
		}
		files = append(files, protocol.FileInfo{
			Name:  e.Name(),
			IsDir: e.IsDir(),
			Size:  size,
		})
	}
	// Dirs first, then files, both alphabetical
	sort.Slice(files, func(i, j int) bool {
		if files[i].IsDir != files[j].IsDir {
			return files[i].IsDir
		}
		return files[i].Name < files[j].Name
	})
	fw.localFiles = files
}

func (fw *FileWindow) localGoUp() {
	parent := filepath.Dir(fw.localPath)
	if parent == fw.localPath {
		return
	}
	fw.localPath = parent
	fw.localPathLbl.SetText(shortPath(fw.localPath))
	fw.loadLocalFiles()
	fw.localList.Refresh()
	fw.localSel = -1
	fw.updateButtons()
}

func (fw *FileWindow) remoteGoUp() {
	if fw.remotePath == "" || fw.remotePath == "/" {
		return
	}
	parent := filepath.Dir(fw.remotePath)
	if parent == "" {
		parent = "/"
	}
	fw.remoteSel = -1
	go fw.conn.RequestFileList(parent)
}

// ─── Button state ─────────────────────────────────────────────────────────────

func (fw *FileWindow) updateButtons() {
	if fw.localSel >= 0 && fw.localSel < len(fw.localFiles) && !fw.localFiles[fw.localSel].IsDir {
		fw.sendBtn.Enable()
	} else {
		fw.sendBtn.Disable()
	}
	if fw.remoteSel >= 0 && fw.remoteSel < len(fw.remoteFiles) && !fw.remoteFiles[fw.remoteSel].IsDir {
		fw.recvBtn.Enable()
	} else {
		fw.recvBtn.Disable()
	}
}

// ─── Transfer actions ─────────────────────────────────────────────────────────

func (fw *FileWindow) doUpload() {
	if fw.localSel < 0 || fw.localSel >= len(fw.localFiles) {
		return
	}
	f := fw.localFiles[fw.localSel]
	if f.IsDir {
		return
	}
	localPath := filepath.Join(fw.localPath, f.Name)
	remoteDir := fw.remotePath

	// Add to log
	tid := fw.addTransfer("↑ Upload", f.Name)

	fw.sendBtn.Disable()
	fw.recvBtn.Disable()
	fw.progressBar.SetValue(0)
	fw.progressBar.Show()
	fw.progressLbl.SetText("Invio: " + f.Name)

	fw.conn.UploadFile(localPath, remoteDir,
		func(transferID, name string, chunks, total int64) {
			fw.updateProgress(tid, name, chunks, total)
		},
		func(transferID, name, savedPath string, err error) {
			fw.finishTransfer(tid, name, err)
			if err != nil {
				dialog.ShowError(fmt.Errorf("upload fallito: %v", err), fw.win)
			} else {
				// Refresh remote list
				go fw.conn.RequestFileList(fw.remotePath)
			}
			fw.sendBtn.Enable()
			fw.updateButtons()
		},
	)
}

func (fw *FileWindow) doDownload() {
	if fw.remoteSel < 0 || fw.remoteSel >= len(fw.remoteFiles) {
		return
	}
	f := fw.remoteFiles[fw.remoteSel]
	if f.IsDir {
		return
	}
	remotePath := fw.remotePath + "/" + f.Name
	localDir := fw.localPath

	tid := fw.addTransfer("↓ Download", f.Name)

	fw.sendBtn.Disable()
	fw.recvBtn.Disable()
	fw.progressBar.SetValue(0)
	fw.progressBar.Show()
	fw.progressLbl.SetText("Ricezione: " + f.Name)

	fw.conn.DownloadFile(remotePath, localDir,
		func(transferID, name string, chunks, total int64) {
			fw.updateProgress(tid, name, chunks, total)
		},
		func(transferID, name, savedPath string, err error) {
			fw.finishTransfer(tid, name, err)
			if err != nil {
				dialog.ShowError(fmt.Errorf("download fallito: %v", err), fw.win)
			} else {
				// Refresh local list
				fw.loadLocalFiles()
				fw.localList.Refresh()
			}
			fw.recvBtn.Enable()
			fw.updateButtons()
		},
	)
}

// ─── Log helpers ──────────────────────────────────────────────────────────────

func (fw *FileWindow) addTransfer(dir, name string) string {
	tid := fmt.Sprintf("%d", time.Now().UnixNano())
	fw.transfersMu.Lock()
	fw.transfers = append(fw.transfers, transferEntry{
		id: tid, name: name, dir: dir, ts: time.Now(),
	})
	fw.transfersMu.Unlock()
	fw.logList.Refresh()
	return tid
}

func (fw *FileWindow) updateProgress(tid, name string, chunks, total int64) {
	fw.transfersMu.Lock()
	for i := range fw.transfers {
		if fw.transfers[i].id == tid {
			fw.transfers[i].chunks = chunks
			fw.transfers[i].total = total
			break
		}
	}
	fw.transfersMu.Unlock()

	var pct float64
	if total > 0 {
		pct = float64(chunks) / float64(total)
	}
	fw.progressBar.SetValue(pct)
	fw.progressLbl.SetText(fmt.Sprintf("%s  %d/%d chunks  (%.0f%%)", name, chunks, total, pct*100))
	fw.logList.Refresh()
}

func (fw *FileWindow) finishTransfer(tid, name string, err error) {
	fw.transfersMu.Lock()
	for i := range fw.transfers {
		if fw.transfers[i].id == tid {
			fw.transfers[i].done = true
			if err != nil {
				fw.transfers[i].errStr = err.Error()
			}
			break
		}
	}
	fw.transfersMu.Unlock()

	fw.progressBar.SetValue(1)
	fw.progressBar.Hide()
	if err != nil {
		fw.progressLbl.SetText("✗ Errore: " + err.Error())
	} else {
		fw.progressLbl.SetText("✓ " + name + " completato")
	}
	fw.logList.Refresh()
}

// ─── Utilities ────────────────────────────────────────────────────────────────

func humanSize(b int64) string {
	switch {
	case b >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(b)/float64(1<<30))
	case b >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(b)/float64(1<<20))
	case b >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(b)/float64(1<<10))
	default:
		return fmt.Sprintf("%d B", b)
	}
}

func shortPath(p string) string {
	if home, err := os.UserHomeDir(); err == nil {
		if len(p) >= len(home) && p[:len(home)] == home {
			return "~" + p[len(home):]
		}
	}
	return p
}
