package ui

import (
	"fmt"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/peardesk/peardesk/pkg/client"
	"github.com/peardesk/peardesk/pkg/config"
	"github.com/peardesk/peardesk/pkg/host"
	"github.com/peardesk/peardesk/pkg/id"
	"github.com/peardesk/peardesk/pkg/relay"
	"github.com/peardesk/peardesk/pkg/tunnel"
)

type MainWindow struct {
	app           fyne.App
	win           fyne.Window
	cfg           *config.Config
	relayClient   *relay.Client
	cloudflaredBin string

	hostServer    *host.Server
	hostTunnel    *tunnel.Tunnel
	hostStatusLbl *widget.Label
	hostIDLbl     *widget.Label

	connectIDEntry   *widget.Entry
	connectPassEntry *widget.Entry
	rememberChk      *widget.Check

	historyList *widget.List
}

func NewMainWindow(app fyne.App, cfg *config.Config) *MainWindow {
	return &MainWindow{
		app:         app,
		cfg:         cfg,
		relayClient: relay.New(cfg.RelayURL),
	}
}

func (mw *MainWindow) Show(cloudflaredBin string) {
	mw.cloudflaredBin = cloudflaredBin
	mw.win = mw.app.NewWindow("PearDesk")
	mw.win.Resize(fyne.NewSize(920, 560))

	tabs := container.NewAppTabs(
		container.NewTabItemWithIcon("Connetti", theme.ComputerIcon(), mw.buildConnectTab()),
		container.NewTabItemWithIcon("Cronologia", theme.HistoryIcon(), mw.buildHistoryTab()),
	)
	tabs.SetTabLocation(container.TabLocationTop)

	mw.win.SetContent(tabs)
	mw.win.SetOnClosed(func() { mw.stopHost() })
	mw.win.ShowAndRun()
}

// ─── Connect Tab ─────────────────────────────────────────────────────────────

func (mw *MainWindow) buildConnectTab() fyne.CanvasObject {
	// HOST PANEL (left)
	mw.hostIDLbl = widget.NewLabel(mw.cfg.HostID)
	mw.hostIDLbl.TextStyle = fyne.TextStyle{Monospace: true, Bold: true}

	copyBtn := widget.NewButtonWithIcon("", theme.ContentCopyIcon(), func() {
		mw.win.Clipboard().SetContent(mw.cfg.HostID)
		dialog.ShowInformation("Copiato", "ID copiato negli appunti", mw.win)
	})

	regenBtn := widget.NewButton("↺  Rigenera ID", func() {
		dialog.ShowConfirm("Rigenera ID",
			"Generare un nuovo ID? Il vecchio ID non funzionerà più.",
			func(ok bool) {
				if !ok {
					return
				}
				newID := id.Generate()
				mw.cfg.HostID = newID
				mw.cfg.Save()
				mw.hostIDLbl.SetText(newID)
				if mw.hostTunnel != nil {
					go mw.relayClient.Register(newID, mw.hostTunnel.URL, mw.cfg.HostPassword)
				}
			}, mw.win)
	})

	passEntry := widget.NewPasswordEntry()
	passEntry.SetPlaceHolder("(nessuna password)")
	passEntry.SetText(mw.cfg.HostPassword)
	passEntry.OnChanged = func(s string) {
		mw.cfg.HostPassword = s
		mw.cfg.Save()
	}

	mw.hostStatusLbl = widget.NewLabel("Pronto — premi Avvia per condividere")
	mw.hostStatusLbl.Wrapping = fyne.TextWrapWord

	startBtn := widget.NewButtonWithIcon("  Avvia condivisione", theme.MediaPlayIcon(), nil)
	startBtn.Importance = widget.HighImportance
	stopBtn := widget.NewButtonWithIcon("  Ferma", theme.MediaStopIcon(), nil)
	stopBtn.Disable()

	startBtn.OnTapped = func() {
		if mw.cloudflaredBin == "" {
			dialog.ShowError(
				fmt.Errorf("cloudflared non trovato.\nInstallalo: https://developers.cloudflare.com/cloudflare-one/connections/connect-apps/install-and-setup/installation/"),
				mw.win,
			)
			return
		}
		startBtn.Disable()
		stopBtn.Enable()
		mw.hostStatusLbl.SetText("Avvio in corso...")
		go mw.startHost(startBtn, stopBtn)
	}
	stopBtn.OnTapped = func() {
		mw.stopHost()
		mw.hostStatusLbl.SetText("Condivisione fermata")
		startBtn.Enable()
		stopBtn.Disable()
	}

	hostCard := widget.NewCard("Il tuo PearDesk", "", container.NewVBox(
		container.NewGridWithColumns(2,
			widget.NewLabel("Il tuo ID:"),
			container.NewHBox(mw.hostIDLbl, copyBtn),
		),
		regenBtn,
		widget.NewSeparator(),
		widget.NewLabel("Password accesso:"),
		passEntry,
		widget.NewSeparator(),
		container.NewHBox(startBtn, stopBtn),
		mw.hostStatusLbl,
	))

	// CLIENT PANEL (right)
	mw.connectIDEntry = widget.NewEntry()
	mw.connectIDEntry.SetPlaceHolder("ABC-123-XYZ")

	mw.connectPassEntry = widget.NewPasswordEntry()
	mw.connectPassEntry.SetPlaceHolder("(se richiesta)")

	mw.rememberChk = widget.NewCheck("Ricorda password per questo host", nil)

	mw.connectIDEntry.OnChanged = func(s string) {
		if pw, ok := mw.cfg.GetHistoryPassword(s); ok {
			mw.connectPassEntry.SetText(pw)
			mw.rememberChk.SetChecked(true)
		}
	}

	connectBtn := widget.NewButtonWithIcon("  Connetti", theme.LoginIcon(), func() {
		hostID := mw.connectIDEntry.Text
		if hostID == "" {
			dialog.ShowError(fmt.Errorf("inserisci l'ID dell'host"), mw.win)
			return
		}
		go mw.connectToHost(hostID, mw.connectPassEntry.Text, mw.rememberChk.Checked)
	})
	connectBtn.Importance = widget.HighImportance

	clientCard := widget.NewCard("Connetti a un host", "", container.NewVBox(
		widget.NewLabel("ID Host:"),
		mw.connectIDEntry,
		widget.NewLabel("Password:"),
		mw.connectPassEntry,
		mw.rememberChk,
		connectBtn,
	))

	split := container.NewHSplit(hostCard, clientCard)
	split.Offset = 0.5
	return split
}

// ─── History Tab ──────────────────────────────────────────────────────────────

func (mw *MainWindow) buildHistoryTab() fyne.CanvasObject {
	mw.historyList = widget.NewList(
		func() int { return len(mw.cfg.History) },
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewLabel(""), // ID
				widget.NewLabel(""), // date
				widget.NewButton("Connetti", nil),
				widget.NewButton("✕", nil),
			)
		},
		func(lid widget.ListItemID, obj fyne.CanvasObject) {
			if lid >= len(mw.cfg.History) {
				return
			}
			row := obj.(*fyne.Container)
			entry := mw.cfg.History[lid]

			idLbl := row.Objects[0].(*widget.Label)
			idLbl.SetText(entry.ID)
			idLbl.TextStyle = fyne.TextStyle{Monospace: true}

			dateLbl := row.Objects[1].(*widget.Label)
			dateLbl.SetText(entry.LastConnected.Format("02/01/2006 15:04"))

			connectBtn := row.Objects[2].(*widget.Button)
			connectBtn.OnTapped = func() {
				pw := entry.Password
				if !entry.RememberPassword {
					pw = ""
				}
				go mw.connectToHost(entry.ID, pw, entry.RememberPassword)
			}

			removeBtn := row.Objects[3].(*widget.Button)
			removeBtn.OnTapped = func() {
				mw.cfg.RemoveHistory(entry.ID)
				mw.cfg.Save()
				mw.historyList.Refresh()
			}
		},
	)

	emptyNote := widget.NewLabelWithStyle(
		"Nessuna connessione nella cronologia.\nConnettersi a un host per aggiungere voci.",
		fyne.TextAlignCenter,
		fyne.TextStyle{Italic: true},
	)

	return container.NewStack(
		container.NewPadded(emptyNote),
		mw.historyList,
	)
}

// ─── Host lifecycle ───────────────────────────────────────────────────────────

func (mw *MainWindow) startHost(startBtn, stopBtn *widget.Button) {
	srv := host.NewServer(mw.cfg.HostPassword)
	srv.OnLog = func(msg string) {
		mw.hostStatusLbl.SetText(msg)
	}
	port, err := srv.Start()
	if err != nil {
		mw.hostStatusLbl.SetText("Errore avvio server: " + err.Error())
		startBtn.Enable()
		stopBtn.Disable()
		return
	}
	mw.hostServer = srv
	mw.hostStatusLbl.SetText(fmt.Sprintf("Server locale avviato (porta %d)\nAvvio tunnel Cloudflare...", port))

	tun, err := tunnel.Start(port, mw.cloudflaredBin)
	if err != nil {
		mw.hostStatusLbl.SetText("Errore tunnel:\n" + err.Error())
		srv.Stop()
		startBtn.Enable()
		stopBtn.Disable()
		return
	}
	mw.hostTunnel = tun
	mw.hostStatusLbl.SetText("Registrazione al relay...")

	if err := mw.relayClient.Register(mw.cfg.HostID, tun.URL, mw.cfg.HostPassword); err != nil {
		mw.hostStatusLbl.SetText(
			"Tunnel attivo (relay non raggiungibile).\n" +
				"URL diretto: " + tun.URL + "\n\n" +
				"I client devono connettersi manualmente.")
	} else {
		mw.hostStatusLbl.SetText(
			"Pronto!\n\n" +
				"Il tuo ID: " + mw.cfg.HostID + "\n\n" +
				"I client possono connettersi con questo ID.")
	}
}

func (mw *MainWindow) stopHost() {
	if mw.hostServer != nil {
		go mw.relayClient.Unregister(mw.cfg.HostID, mw.cfg.HostPassword)
		mw.hostServer.Stop()
		mw.hostServer = nil
	}
	if mw.hostTunnel != nil {
		mw.hostTunnel.Stop()
		mw.hostTunnel = nil
	}
}

// ─── Client connect ───────────────────────────────────────────────────────────

func (mw *MainWindow) connectToHost(hostID, password string, remember bool) {
	pd := dialog.NewProgress("Connessione", "Ricerca host "+hostID+"…", mw.win)
	pd.Show()

	pd.SetValue(0.2)
	tunnelURL, err := mw.relayClient.Lookup(hostID)
	if err != nil {
		pd.Hide()
		dialog.ShowError(fmt.Errorf("host non trovato: %v", err), mw.win)
		return
	}

	pd.SetValue(0.6)
	time.Sleep(200 * time.Millisecond)

	conn, err := client.Connect(tunnelURL, password)
	if err != nil {
		pd.Hide()
		dialog.ShowError(err, mw.win)
		return
	}

	pd.SetValue(1.0)
	pd.Hide()

	mw.cfg.AddOrUpdateHistory(hostID, hostID, password, remember)
	mw.cfg.Save()
	if mw.historyList != nil {
		mw.historyList.Refresh()
	}

	ShowRemoteWindow(mw.app, conn, hostID)
}
