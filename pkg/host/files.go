package host

import (
	"os"
	"path/filepath"
	"time"

	"github.com/peardesk/peardesk/pkg/protocol"
)

func listFiles(path string) []protocol.FileInfo {
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			path = "/"
		} else {
			path = home
		}
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return []protocol.FileInfo{}
	}
	files := make([]protocol.FileInfo, 0, len(entries))
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			continue
		}
		files = append(files, protocol.FileInfo{
			Name:    e.Name(),
			Size:    info.Size(),
			IsDir:   e.IsDir(),
			ModTime: info.ModTime().Format(time.RFC3339),
		})
	}
	return files
}

func GetFilePath(base, rel string) string {
	return filepath.Join(base, rel)
}
