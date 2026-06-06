package host

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/peardesk/peardesk/pkg/protocol"
)

type Server struct {
	port    int
	password string
	httpSrv *http.Server
	OnLog   func(string)
}

var upgrader = websocket.Upgrader{
	CheckOrigin:     func(r *http.Request) bool { return true },
	ReadBufferSize:  1024 * 1024,
	WriteBufferSize: 1024 * 1024,
}

func NewServer(password string) *Server {
	return &Server{password: password}
}

func (s *Server) Start() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	s.port = listener.Addr().(*net.TCPAddr).Port

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", s.handleWS)
	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("pong"))
	})

	s.httpSrv = &http.Server{Handler: mux}
	go func() {
		if err := s.httpSrv.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Printf("host server error: %v", err)
		}
	}()
	return s.port, nil
}

func (s *Server) Port() int { return s.port }

func (s *Server) Stop() {
	if s.httpSrv != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		s.httpSrv.Shutdown(ctx)
	}
}

func (s *Server) logf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	if s.OnLog != nil {
		s.OnLog(msg)
	} else {
		log.Println(msg)
	}
}

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	remoteAddr := r.RemoteAddr
	s.logf("Nuova connessione da %s", remoteAddr)

	// ── Auth ─────────────────────────────────────────────────────────────────
	conn.SetReadDeadline(time.Now().Add(15 * time.Second))
	_, msgBytes, err := conn.ReadMessage()
	if err != nil {
		s.logf("Errore lettura auth: %v", err)
		return
	}
	conn.SetReadDeadline(time.Time{})

	var authMsg protocol.AuthMsg
	if err := json.Unmarshal(msgBytes, &authMsg); err != nil || authMsg.Type != protocol.TypeAuth {
		conn.WriteJSON(protocol.AuthResultMsg{Type: protocol.TypeAuthFail})
		return
	}
	if s.password != "" && authMsg.Password != s.password {
		conn.WriteJSON(protocol.AuthResultMsg{Type: protocol.TypeAuthFail})
		s.logf("Password errata da %s", remoteAddr)
		return
	}
	conn.WriteJSON(protocol.AuthResultMsg{Type: protocol.TypeAuthOK})
	s.logf("Client autenticato: %s", remoteAddr)

	screenW, screenH := screenSize()

	frameTicker := time.NewTicker(33 * time.Millisecond)
	defer frameTicker.Stop()

	done := make(chan struct{})

	// Track in-progress uploads keyed by transfer_id
	uploads := make(map[string]*incomingUpload)

	// ── Read loop ─────────────────────────────────────────────────────────────
	go func() {
		defer close(done)
		for {
			_, msgBytes, err := conn.ReadMessage()
			if err != nil {
				return
			}
			var msg protocol.Message
			if err := json.Unmarshal(msgBytes, &msg); err != nil {
				continue
			}
			switch msg.Type {

			// Input events
			case protocol.TypeMouseEvent:
				var me protocol.MouseEventMsg
				if err := json.Unmarshal(msgBytes, &me); err == nil {
					x := int(me.X * float64(screenW))
					y := int(me.Y * float64(screenH))
					switch me.Action {
					case "move":
						injectMouseMove(x, y)
					case "down":
						injectMouseClick(x, y, me.Button, true)
					case "up":
						injectMouseClick(x, y, me.Button, false)
					case "scroll":
						injectMouseScroll(x, y, me.ScrollY)
					}
				}
			case protocol.TypeKeyEvent:
				var ke protocol.KeyEventMsg
				if err := json.Unmarshal(msgBytes, &ke); err == nil {
					injectKeyEvent(ke.Key, ke.Action == "down", ke.Modifiers)
				}

			// Keep-alive
			case protocol.TypePing:
				conn.WriteJSON(protocol.PongMsg{Type: protocol.TypePong})

			// File browsing
			case protocol.TypeFileListReq:
				var req protocol.FileListReqMsg
				if err := json.Unmarshal(msgBytes, &req); err == nil {
					conn.WriteJSON(protocol.FileListMsg{
						Type:  protocol.TypeFileList,
						Path:  req.Path,
						Files: listFiles(req.Path),
					})
				}

			// Download: client requests a file from host
			case protocol.TypeFileDownloadReq:
				go s.handleDownload(conn, msgBytes)

			// Upload: client announces incoming file
			case protocol.TypeFileUploadStart:
				s.handleUploadStart(conn, uploads, msgBytes)

			// Upload: client sends a chunk
			case protocol.TypeFileChunk:
				s.handleUploadChunk(conn, uploads, msgBytes)
			}
		}
	}()

	// ── Frame send loop ───────────────────────────────────────────────────────
	for {
		select {
		case <-done:
			s.logf("Client disconnesso: %s", remoteAddr)
			return
		case <-frameTicker.C:
			b64, w, h, err := captureScreen(1920, 1080)
			if err != nil || b64 == "" {
				continue
			}
			if err := conn.WriteJSON(protocol.FrameMsg{
				Type:   protocol.TypeFrame,
				Data:   b64,
				Width:  w,
				Height: h,
			}); err != nil {
				return
			}
		}
	}
}
