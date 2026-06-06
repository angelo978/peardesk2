package host

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gorilla/websocket"
	"github.com/peardesk/peardesk/pkg/protocol"
	"github.com/peardesk/peardesk/pkg/transfer"
)

// handleDownload streams a local file to the client in chunks.
func (s *Server) handleDownload(conn *websocket.Conn, reqBytes []byte) {
	var req protocol.FileDownloadReqMsg
	if err := json.Unmarshal(reqBytes, &req); err != nil {
		conn.WriteJSON(protocol.FileTransferErrMsg{
			Type: protocol.TypeFileTransferErr, TransferID: "", Error: "bad request",
		})
		return
	}

	info, err := os.Stat(req.Path)
	if err != nil {
		conn.WriteJSON(protocol.FileTransferErrMsg{
			Type: protocol.TypeFileTransferErr, TransferID: req.TransferID,
			Error: fmt.Sprintf("file non trovato: %v", err),
		})
		return
	}
	if info.IsDir() {
		conn.WriteJSON(protocol.FileTransferErrMsg{
			Type: protocol.TypeFileTransferErr, TransferID: req.TransferID,
			Error: "impossibile scaricare una cartella",
		})
		return
	}

	fileSize := info.Size()
	total := transfer.TotalChunks(fileSize)
	name := filepath.Base(req.Path)
	s.logf("Download richiesto: %s (%d byte, %d chunks)", name, fileSize, total)

	for i := int64(0); i < total; i++ {
		b64, err := transfer.ChunkFile(req.Path, i)
		if err != nil {
			conn.WriteJSON(protocol.FileTransferErrMsg{
				Type: protocol.TypeFileTransferErr, TransferID: req.TransferID,
				Error: fmt.Sprintf("lettura chunk %d: %v", i, err),
			})
			return
		}
		msg := protocol.FileChunkMsg{
			Type:       protocol.TypeFileChunk,
			TransferID: req.TransferID,
			Name:       name,
			Index:      i,
			Total:      total,
			FileSize:   fileSize,
			Data:       b64,
		}
		if err := conn.WriteJSON(msg); err != nil {
			s.logf("Errore invio chunk %d: %v", i, err)
			return
		}
	}

	conn.WriteJSON(protocol.FileTransferDoneMsg{
		Type: protocol.TypeFileTransferDone, TransferID: req.TransferID, Name: name,
	})
	s.logf("Download completato: %s", name)
}

// incomingUpload tracks an in-progress upload from the client.
type incomingUpload struct {
	writer   *transfer.Writer
	name     string
	total    int64
	received int64
}

// handleUploadStart acknowledges the start of an upload from the client.
func (s *Server) handleUploadStart(
	conn *websocket.Conn, uploads map[string]*incomingUpload, reqBytes []byte,
) {
	var req protocol.FileUploadStartMsg
	if err := json.Unmarshal(reqBytes, &req); err != nil {
		return
	}

	destDir := req.DestPath
	if destDir == "" {
		home, _ := os.UserHomeDir()
		destDir = home
	}
	if err := os.MkdirAll(destDir, 0755); err != nil {
		conn.WriteJSON(protocol.FileTransferErrMsg{
			Type: protocol.TypeFileTransferErr, TransferID: req.TransferID,
			Error: "impossibile creare cartella destinazione: " + err.Error(),
		})
		return
	}

	destPath := filepath.Join(destDir, req.Name)
	// Avoid overwriting: add suffix if file exists
	if _, err := os.Stat(destPath); err == nil {
		ext := filepath.Ext(req.Name)
		base := req.Name[:len(req.Name)-len(ext)]
		destPath = filepath.Join(destDir, fmt.Sprintf("%s_1%s", base, ext))
	}

	w, err := transfer.NewWriter(destPath, req.Total)
	if err != nil {
		conn.WriteJSON(protocol.FileTransferErrMsg{
			Type: protocol.TypeFileTransferErr, TransferID: req.TransferID,
			Error: "impossibile creare file: " + err.Error(),
		})
		return
	}

	uploads[req.TransferID] = &incomingUpload{
		writer: w, name: req.Name, total: req.Total,
	}

	conn.WriteJSON(protocol.FileUploadReadyMsg{
		Type: protocol.TypeFileUploadReady, TransferID: req.TransferID,
	})
	s.logf("Upload pronto: %s → %s (%d chunks)", req.Name, destPath, req.Total)
}

// handleUploadChunk writes an incoming chunk to disk.
func (s *Server) handleUploadChunk(
	conn *websocket.Conn, uploads map[string]*incomingUpload, chunkBytes []byte,
) {
	var chunk protocol.FileChunkMsg
	if err := json.Unmarshal(chunkBytes, &chunk); err != nil {
		return
	}

	up, ok := uploads[chunk.TransferID]
	if !ok {
		return
	}

	_, done, err := up.writer.WriteChunk(chunk.Data)
	if err != nil {
		s.logf("Errore scrittura chunk: %v", err)
		up.writer.Abort()
		delete(uploads, chunk.TransferID)
		conn.WriteJSON(protocol.FileTransferErrMsg{
			Type: protocol.TypeFileTransferErr, TransferID: chunk.TransferID,
			Error: err.Error(),
		})
		return
	}

	up.received++

	if done {
		savedPath := up.writer.Path()
		delete(uploads, chunk.TransferID)
		conn.WriteJSON(protocol.FileUploadDoneMsg{
			Type:       protocol.TypeFileUploadDone,
			TransferID: chunk.TransferID,
			SavedPath:  savedPath,
		})
		s.logf("Upload completato: %s → %s", up.name, savedPath)
	}
}
