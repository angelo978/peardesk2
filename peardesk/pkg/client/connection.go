package client

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/peardesk/peardesk/pkg/protocol"
	"github.com/peardesk/peardesk/pkg/transfer"
)

// TransferProgress is called periodically during a transfer.
type TransferProgress func(transferID, name string, chunks, total int64)

// TransferDone is called when a transfer finishes (success or error).
type TransferDone func(transferID, name string, savedPath string, err error)

// downloadState tracks an in-progress file download or upload-ack.
type downloadState struct {
	writer   *transfer.Writer // nil until first chunk arrives (download) or unused (upload-ack)
	name     string           // filename (or destPath before first chunk for downloads)
	destDir  string           // local destination directory (downloads only)
	total    int64
	received int64
	onProg   TransferProgress
	onDone   TransferDone
}

// Connection wraps a WebSocket connection to a PearDesk host.
type Connection struct {
	conn      *websocket.Conn
	tunnelURL string
	mu        sync.Mutex // guards conn writes

	OnFrame    func(b64 string, w, h int)
	OnFileList func(path string, files []protocol.FileInfo)
	OnLog      func(string)
	OnError    func(error)
	OnClose    func()

	transfers   map[string]*downloadState
	transfersMu sync.Mutex
}

// Connect dials the host and authenticates.
func Connect(tunnelURL, password string) (*Connection, error) {
	wsURL := toWSURL(tunnelURL) + "/ws"
	dialer := websocket.Dialer{
		HandshakeTimeout: 15 * time.Second,
		ReadBufferSize:   1 << 20,
		WriteBufferSize:  1 << 20,
	}
	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("connessione fallita: %w", err)
	}

	if err := conn.WriteJSON(protocol.AuthMsg{Type: protocol.TypeAuth, Password: password}); err != nil {
		conn.Close()
		return nil, fmt.Errorf("errore invio auth: %w", err)
	}

	conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	_, msgBytes, err := conn.ReadMessage()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("errore lettura risposta auth: %w", err)
	}
	conn.SetReadDeadline(time.Time{})

	var result protocol.AuthResultMsg
	if err := json.Unmarshal(msgBytes, &result); err != nil || result.Type == protocol.TypeAuthFail {
		conn.Close()
		return nil, fmt.Errorf("password errata")
	}

	c := &Connection{
		conn:      conn,
		tunnelURL: tunnelURL,
		transfers: make(map[string]*downloadState),
	}
	go c.readLoop()
	return c, nil
}

func (c *Connection) writeMsg(v interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn.WriteJSON(v)
}

// ─── Read loop ────────────────────────────────────────────────────────────────

func (c *Connection) readLoop() {
	defer func() {
		c.conn.Close()
		if c.OnClose != nil {
			c.OnClose()
		}
	}()
	for {
		_, msgBytes, err := c.conn.ReadMessage()
		if err != nil {
			if c.OnError != nil && !websocket.IsCloseError(err,
				websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				c.OnError(err)
			}
			return
		}
		var msg protocol.Message
		if err := json.Unmarshal(msgBytes, &msg); err != nil {
			continue
		}
		switch msg.Type {
		case protocol.TypeFrame:
			var f protocol.FrameMsg
			if err := json.Unmarshal(msgBytes, &f); err == nil && c.OnFrame != nil {
				c.OnFrame(f.Data, f.Width, f.Height)
			}
		case protocol.TypeFileList:
			var fl protocol.FileListMsg
			if err := json.Unmarshal(msgBytes, &fl); err == nil && c.OnFileList != nil {
				c.OnFileList(fl.Path, fl.Files)
			}
		case protocol.TypeFileChunk:
			c.handleDownloadChunk(msgBytes)
		case protocol.TypeFileTransferDone:
			// Already handled inside handleDownloadChunk when done==true
		case protocol.TypeFileTransferErr:
			var e protocol.FileTransferErrMsg
			if err := json.Unmarshal(msgBytes, &e); err == nil {
				c.transfersMu.Lock()
				if ds, ok := c.transfers[e.TransferID]; ok {
					if ds.writer != nil {
						ds.writer.Abort()
					}
					if ds.onDone != nil {
						go ds.onDone(e.TransferID, ds.name, "", fmt.Errorf("%s", e.Error))
					}
					delete(c.transfers, e.TransferID)
				}
				c.transfersMu.Unlock()
			}
		case protocol.TypeFileUploadReady:
			// Upload-ready: the host has prepared to receive; uploading starts in UploadFile goroutine.
		case protocol.TypeFileUploadDone:
			var d protocol.FileUploadDoneMsg
			if err := json.Unmarshal(msgBytes, &d); err == nil {
				c.transfersMu.Lock()
				if ds, ok := c.transfers[d.TransferID]; ok {
					if ds.onDone != nil {
						go ds.onDone(d.TransferID, ds.name, d.SavedPath, nil)
					}
					delete(c.transfers, d.TransferID)
				}
				c.transfersMu.Unlock()
			}
		case protocol.TypePong:
			// ignore
		}
	}
}

// ─── File browsing ────────────────────────────────────────────────────────────

func (c *Connection) RequestFileList(path string) {
	c.writeMsg(protocol.FileListReqMsg{Type: protocol.TypeFileListReq, Path: path})
}

// ─── Download (request file from host) ───────────────────────────────────────

// DownloadFile asks the host to send remotePath; saves it into localDir.
func (c *Connection) DownloadFile(
	remotePath, localDir string,
	onProgress TransferProgress,
	onDone TransferDone,
) string {
	tid := newTransferID()
	ds := &downloadState{
		name:    filepath.Base(remotePath),
		destDir: localDir,
		onProg:  onProgress,
		onDone:  onDone,
	}
	c.transfersMu.Lock()
	c.transfers[tid] = ds
	c.transfersMu.Unlock()

	c.writeMsg(protocol.FileDownloadReqMsg{
		Type:       protocol.TypeFileDownloadReq,
		TransferID: tid,
		Path:       remotePath,
	})
	return tid
}

func (c *Connection) handleDownloadChunk(msgBytes []byte) {
	var chunk protocol.FileChunkMsg
	if err := json.Unmarshal(msgBytes, &chunk); err != nil {
		return
	}
	c.transfersMu.Lock()
	ds, ok := c.transfers[chunk.TransferID]
	if !ok {
		c.transfersMu.Unlock()
		return
	}

	// Create writer lazily on the first chunk.
	if ds.writer == nil {
		ds.total = chunk.Total
		ds.name = chunk.Name
		destPath := filepath.Join(ds.destDir, chunk.Name)
		// Avoid overwriting
		if _, err := os.Stat(destPath); err == nil {
			ext := filepath.Ext(chunk.Name)
			base := chunk.Name[:len(chunk.Name)-len(ext)]
			destPath = filepath.Join(ds.destDir, fmt.Sprintf("%s_1%s", base, ext))
		}
		w, err := transfer.NewWriter(destPath, chunk.Total)
		if err != nil {
			c.transfersMu.Unlock()
			if ds.onDone != nil {
				go ds.onDone(chunk.TransferID, chunk.Name, "", err)
			}
			delete(c.transfers, chunk.TransferID)
			return
		}
		ds.writer = w
	}
	c.transfersMu.Unlock()

	_, done, err := ds.writer.WriteChunk(chunk.Data)
	if err != nil {
		ds.writer.Abort()
		c.transfersMu.Lock()
		delete(c.transfers, chunk.TransferID)
		c.transfersMu.Unlock()
		if ds.onDone != nil {
			go ds.onDone(chunk.TransferID, ds.name, "", err)
		}
		return
	}

	ds.received++
	if ds.onProg != nil {
		go ds.onProg(chunk.TransferID, ds.name, ds.received, chunk.Total)
	}
	if done {
		savedPath := ds.writer.Path()
		c.transfersMu.Lock()
		delete(c.transfers, chunk.TransferID)
		c.transfersMu.Unlock()
		if ds.onDone != nil {
			go ds.onDone(chunk.TransferID, ds.name, savedPath, nil)
		}
	}
}

// ─── Upload (send local file to host) ────────────────────────────────────────

// UploadFile sends localPath to remoteDestDir on the host.
func (c *Connection) UploadFile(
	localPath, remoteDestDir string,
	onProgress TransferProgress,
	onDone TransferDone,
) {
	go func() {
		info, err := os.Stat(localPath)
		if err != nil {
			if onDone != nil {
				onDone("", filepath.Base(localPath), "", err)
			}
			return
		}
		name := info.Name()
		fileSize := info.Size()
		totalChunks := transfer.TotalChunks(fileSize)
		tid := newTransferID()

		// Register state so TypeFileUploadDone can call onDone.
		ds := &downloadState{
			name:   name,
			total:  totalChunks,
			onDone: onDone,
		}
		c.transfersMu.Lock()
		c.transfers[tid] = ds
		c.transfersMu.Unlock()

		// Announce upload
		if err := c.writeMsg(protocol.FileUploadStartMsg{
			Type:       protocol.TypeFileUploadStart,
			TransferID: tid,
			Name:       name,
			FileSize:   fileSize,
			Total:      totalChunks,
			DestPath:   remoteDestDir,
		}); err != nil {
			if onDone != nil {
				onDone(tid, name, "", err)
			}
			return
		}

		// Wait for host to be ready (TypeFileUploadReady arrives asynchronously).
		time.Sleep(400 * time.Millisecond)

		// Stream chunks
		for i := int64(0); i < totalChunks; i++ {
			b64, err := transfer.ChunkFile(localPath, i)
			if err != nil {
				if onDone != nil {
					onDone(tid, name, "", fmt.Errorf("lettura chunk %d: %w", i, err))
				}
				return
			}
			if err := c.writeMsg(protocol.FileChunkMsg{
				Type:       protocol.TypeFileChunk,
				TransferID: tid,
				Name:       name,
				Index:      i,
				Total:      totalChunks,
				FileSize:   fileSize,
				Data:       b64,
			}); err != nil {
				if onDone != nil {
					onDone(tid, name, "", fmt.Errorf("invio chunk %d: %w", i, err))
				}
				return
			}
			if onProgress != nil {
				onProgress(tid, name, i+1, totalChunks)
			}
		}
		// onDone fires when TypeFileUploadDone arrives from host (or TypeFileTransferErr).
	}()
}

// ─── Input events ─────────────────────────────────────────────────────────────

func (c *Connection) SendMouseMove(xR, yR float64) {
	c.writeMsg(protocol.MouseEventMsg{Type: protocol.TypeMouseEvent, X: xR, Y: yR, Action: "move"})
}
func (c *Connection) SendMouseDown(xR, yR float64, button string) {
	c.writeMsg(protocol.MouseEventMsg{Type: protocol.TypeMouseEvent, X: xR, Y: yR, Button: button, Action: "down"})
}
func (c *Connection) SendMouseUp(xR, yR float64, button string) {
	c.writeMsg(protocol.MouseEventMsg{Type: protocol.TypeMouseEvent, X: xR, Y: yR, Button: button, Action: "up"})
}
func (c *Connection) SendScroll(xR, yR, dy float64) {
	c.writeMsg(protocol.MouseEventMsg{Type: protocol.TypeMouseEvent, X: xR, Y: yR, Action: "scroll", ScrollY: dy})
}
func (c *Connection) SendKeyDown(key string, mods []string) {
	c.writeMsg(protocol.KeyEventMsg{Type: protocol.TypeKeyEvent, Key: key, Action: "down", Modifiers: mods})
}
func (c *Connection) SendKeyUp(key string, mods []string) {
	c.writeMsg(protocol.KeyEventMsg{Type: protocol.TypeKeyEvent, Key: key, Action: "up", Modifiers: mods})
}
func (c *Connection) Ping() {
	c.writeMsg(protocol.PingMsg{Type: protocol.TypePing})
}
func (c *Connection) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.conn.WriteMessage(websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	c.conn.Close()
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func toWSURL(httpURL string) string {
	u, err := url.Parse(httpURL)
	if err != nil {
		return strings.Replace(httpURL, "https://", "wss://", 1)
	}
	switch u.Scheme {
	case "https":
		u.Scheme = "wss"
	case "http":
		u.Scheme = "ws"
	}
	return strings.TrimRight(u.String(), "/")
}

var (
	tidMu  sync.Mutex
	tidSeq int64
)

func newTransferID() string {
	tidMu.Lock()
	defer tidMu.Unlock()
	tidSeq++
	return fmt.Sprintf("tid-%d-%d", time.Now().UnixNano(), tidSeq)
}
