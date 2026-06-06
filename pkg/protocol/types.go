package protocol

const (
        TypeFrame      = "frame"
        TypeMouseEvent = "mouse"
        TypeKeyEvent   = "key"
        TypeClipboard  = "clipboard"
        TypeAuth       = "auth"
        TypeAuthOK     = "auth_ok"
        TypeAuthFail   = "auth_fail"
        TypePing       = "ping"
        TypePong       = "pong"

        // File browsing
        TypeFileListReq = "file_list_req"
        TypeFileList    = "file_list"

        // Download: client requests → host sends chunks
        TypeFileDownloadReq  = "file_download_req"
        TypeFileChunk        = "file_chunk"
        TypeFileTransferDone = "file_transfer_done"
        TypeFileTransferErr  = "file_transfer_err"

        // Upload: client starts → host acks → client sends chunks → host confirms
        TypeFileUploadStart = "file_upload_start"
        TypeFileUploadReady = "file_upload_ready"
        TypeFileUploadDone  = "file_upload_done"
)

// ─── Auth ────────────────────────────────────────────────────────────────────

type Message struct {
        Type string `json:"type"`
}

type AuthMsg struct {
        Type     string `json:"type"`
        Password string `json:"password"`
}

type AuthResultMsg struct {
        Type string `json:"type"`
}

// ─── Video streaming ─────────────────────────────────────────────────────────

type FrameMsg struct {
        Type   string `json:"type"`
        Data   string `json:"data"` // base64 JPEG
        Width  int    `json:"width"`
        Height int    `json:"height"`
}

// ─── Input ───────────────────────────────────────────────────────────────────

type MouseEventMsg struct {
        Type    string  `json:"type"`
        X       float64 `json:"x"`
        Y       float64 `json:"y"`
        Button  string  `json:"button"`  // "left","right","middle","none"
        Action  string  `json:"action"`  // "move","down","up","scroll"
        ScrollY float64 `json:"scroll_y,omitempty"`
}

type KeyEventMsg struct {
        Type      string   `json:"type"`
        Key       string   `json:"key"`
        Action    string   `json:"action"` // "down","up"
        Modifiers []string `json:"modifiers,omitempty"`
}

type PingMsg struct{ Type string `json:"type"` }
type PongMsg struct{ Type string `json:"type"` }

// ClipboardMsg is sent in both directions when clipboard content changes.
type ClipboardMsg struct {
        Type string `json:"type"`
        Text string `json:"text"`
}

// ─── File browsing ────────────────────────────────────────────────────────────

type FileListReqMsg struct {
        Type string `json:"type"`
        Path string `json:"path"`
}

type FileInfo struct {
        Name    string `json:"name"`
        Size    int64  `json:"size"`
        IsDir   bool   `json:"is_dir"`
        ModTime string `json:"mod_time"`
}

type FileListMsg struct {
        Type  string     `json:"type"`
        Path  string     `json:"path"`
        Files []FileInfo `json:"files"`
}

// ─── Download (host → client) ─────────────────────────────────────────────────

// Client → Host: request a file download
type FileDownloadReqMsg struct {
        Type       string `json:"type"`
        TransferID string `json:"transfer_id"`
        Path       string `json:"path"` // full remote path
}

// Host → Client (repeated): one chunk of file data
type FileChunkMsg struct {
        Type       string `json:"type"`
        TransferID string `json:"transfer_id"`
        Name       string `json:"name"`
        Index      int64  `json:"index"`
        Total      int64  `json:"total"`
        FileSize   int64  `json:"file_size"`
        Data       string `json:"data"` // base64-encoded chunk bytes
}

// Either direction: signals successful completion
type FileTransferDoneMsg struct {
        Type       string `json:"type"`
        TransferID string `json:"transfer_id"`
        Name       string `json:"name"`
}

// Either direction: signals an error
type FileTransferErrMsg struct {
        Type       string `json:"type"`
        TransferID string `json:"transfer_id"`
        Error      string `json:"error"`
}

// ─── Upload (client → host) ───────────────────────────────────────────────────

// Client → Host: announce an incoming upload
type FileUploadStartMsg struct {
        Type       string `json:"type"`
        TransferID string `json:"transfer_id"`
        Name       string `json:"name"`
        FileSize   int64  `json:"file_size"`
        Total      int64  `json:"total"` // total chunks
        DestPath   string `json:"dest_path"` // remote destination directory
}

// Host → Client: ready to receive
type FileUploadReadyMsg struct {
        Type       string `json:"type"`
        TransferID string `json:"transfer_id"`
}

// Host → Client: upload complete and saved
type FileUploadDoneMsg struct {
        Type       string `json:"type"`
        TransferID string `json:"transfer_id"`
        SavedPath  string `json:"saved_path"`
}
