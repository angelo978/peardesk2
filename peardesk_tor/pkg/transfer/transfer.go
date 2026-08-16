// Package transfer provides chunking/dechunking utilities for file transfers.
package transfer

import (
	"encoding/base64"
	"fmt"
	"io"
	"os"
)

const ChunkSize = 256 * 1024 // 256 KB per chunk

// TotalChunks returns the number of chunks needed for a given file size.
func TotalChunks(fileSize int64) int64 {
	if fileSize == 0 {
		return 1
	}
	chunks := fileSize / ChunkSize
	if fileSize%ChunkSize != 0 {
		chunks++
	}
	return chunks
}

// ChunkFile reads chunk[index] from localPath and returns it as a base64 string.
func ChunkFile(localPath string, index int64) (string, error) {
	f, err := os.Open(localPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	if _, err := f.Seek(index*ChunkSize, io.SeekStart); err != nil {
		return "", err
	}

	buf := make([]byte, ChunkSize)
	n, err := f.Read(buf)
	if err != nil && err != io.EOF {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(buf[:n]), nil
}

// Writer assembles incoming file chunks and writes them to disk.
type Writer struct {
	path     string
	f        *os.File
	received int64
	total    int64
}

// NewWriter creates a new Writer that will write to destPath.
// total is the expected number of chunks.
func NewWriter(destPath string, total int64) (*Writer, error) {
	f, err := os.Create(destPath)
	if err != nil {
		return nil, fmt.Errorf("crea file %q: %w", destPath, err)
	}
	return &Writer{path: destPath, f: f, total: total}, nil
}

// WriteChunk decodes b64Data and appends it to the file.
// Returns (bytesWritten, done, error). When done==true the file is closed.
func (w *Writer) WriteChunk(b64Data string) (int, bool, error) {
	data, err := base64.StdEncoding.DecodeString(b64Data)
	if err != nil {
		return 0, false, fmt.Errorf("base64 decode: %w", err)
	}
	n, err := w.f.Write(data)
	if err != nil {
		return n, false, err
	}
	w.received++
	done := w.received >= w.total
	if done {
		w.f.Close()
	}
	return n, done, nil
}

// Abort closes and removes the partially written file.
func (w *Writer) Abort() {
	if w.f != nil {
		w.f.Close()
	}
	os.Remove(w.path)
}

// Path returns the destination file path.
func (w *Writer) Path() string {
	return w.path
}
