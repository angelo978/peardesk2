package host

import (
        "context"
        "encoding/json"
        "fmt"
        "log"
        "net"
        "net/http"
        "sync"
        "time"

        "github.com/gorilla/websocket"
        "github.com/peardesk/peardesk/pkg/clipboard"
        "github.com/peardesk/peardesk/pkg/protocol"
)

type Server struct {
        port     int
        password string
        httpSrv  *http.Server
        OnLog    func(string)
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
        port := listener.Addr().(*net.TCPAddr).Port

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
        return port, nil
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

        uploads := make(map[string]*incomingUpload)

        // ── Clipboard monitor (host → client) ─────────────────────────────────────
        // Channel to serialize writes to conn from multiple goroutines.
        writeMu := &sync.Mutex{}
        safeWrite := func(v interface{}) error {
                writeMu.Lock()
                defer writeMu.Unlock()
                return conn.WriteJSON(v)
        }

        cbMon := clipboard.New(500 * time.Millisecond)
        cbMon.Start(func(text string) {
                safeWrite(protocol.ClipboardMsg{Type: protocol.TypeClipboard, Text: text})
                s.logf("Clipboard → client (%d chars)", len(text))
        })
        defer cbMon.Stop()

        done := make(chan struct{})

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

                        case protocol.TypePing:
                                safeWrite(protocol.PongMsg{Type: protocol.TypePong})

                        // Direct character input (handles uppercase, @, F-keys, etc.)
                        case protocol.TypeRune:
                                var rm protocol.RuneMsg
                                if err := json.Unmarshal(msgBytes, &rm); err == nil && rm.Text != "" {
                                        injectRune(rm.Text)
                                }

                        // Clipboard: client changed their clipboard → apply on host
                        case protocol.TypeClipboard:
                                var cb protocol.ClipboardMsg
                                if err := json.Unmarshal(msgBytes, &cb); err == nil && cb.Text != "" {
                                        if err := cbMon.Write(cb.Text); err == nil {
                                                s.logf("Clipboard ← client (%d chars)", len(cb.Text))
                                        }
                                }

                        case protocol.TypeFileListReq:
                                var req protocol.FileListReqMsg
                                if err := json.Unmarshal(msgBytes, &req); err == nil {
                                        safeWrite(protocol.FileListMsg{
                                                Type:  protocol.TypeFileList,
                                                Path:  req.Path,
                                                Files: listFiles(req.Path),
                                        })
                                }

                        case protocol.TypeFileDownloadReq:
                                go s.handleDownload(conn, writeMu, msgBytes)

                        case protocol.TypeFileUploadStart:
                                s.handleUploadStart(conn, writeMu, uploads, msgBytes)

                        case protocol.TypeFileChunk:
                                s.handleUploadChunk(conn, writeMu, uploads, msgBytes)
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
                        if err := safeWrite(protocol.FrameMsg{
                                Type: protocol.TypeFrame, Data: b64, Width: w, Height: h,
                        }); err != nil {
                                return
                        }
                }
        }
}
