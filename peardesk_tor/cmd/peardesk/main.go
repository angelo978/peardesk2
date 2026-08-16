package main

import (
	"log"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"

	"github.com/peardesk/peardesk/pkg/config"
	"github.com/peardesk/peardesk/pkg/id"
	"github.com/peardesk/peardesk/pkg/tunnel"
	"github.com/peardesk/peardesk/pkg/ui"
)

var Version = "1.0.0"

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Errore caricamento configurazione: %v", err)
	}

	if cfg.HostID == "" {
		cfg.HostID = id.Generate()
		cfg.Save()
	}

	home, _ := os.UserHomeDir()
	dataDir := filepath.Join(home, ".peardesk")

	// Auto-download cloudflared if not available; show progress to stdout only
	cloudflaredBin, cfErr := tunnel.FindOrDownloadCloudflared(dataDir, func(msg string) {
		log.Println(msg)
	})
	if cfErr != nil {
		log.Printf("Avviso cloudflared: %v", cfErr)
		cloudflaredBin = ""
	}

	a := app.NewWithID("com.peardesk.app")

	iconRes := loadIcon()
	if iconRes != nil {
		a.SetIcon(iconRes)
	}

	mw := ui.NewMainWindow(a, cfg)
	mw.Show(cloudflaredBin)
}

func loadIcon() fyne.Resource {
	candidates := []string{
		"assets/icon.png",
		filepath.Join(filepath.Dir(os.Args[0]), "icon.png"),
	}
	for _, p := range candidates {
		if data, err := os.ReadFile(p); err == nil {
			return &fyne.StaticResource{StaticName: "icon.png", StaticContent: data}
		}
	}
	return nil
}
