package ui

import (
	"fmt"
	"os"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/peardesk/peardesk/pkg/autostart"
	"github.com/peardesk/peardesk/pkg/client"
	"github.com/peardesk/peardesk/pkg/config"
	"github.com/peardesk/peardesk/pkg/host"
	"github.com/peardesk/peardesk/pkg/i18n"
	"github.com/peardesk/peardesk/pkg/id"
	"github.com/peardesk/peardesk/pkg/relay"
	"github.com/peardesk/peardesk/pkg/tunnel"
)

type MainWindow struct {
	app            fyne.App
	win            fyne.Window
	cfg            *config.Config
	relayClient    *relay.Client
	cloudflaredBin string

	hostServer *host.Server
	hostTunnel *tunnel.Tunnel

	hostIDLbl     *widget.Label
	hostStatusLbl *widget.Label

	connectIDEntry   *widget.Entry
	connectPassEntry *widget.Entry
	rememberChk      *widget.Check

	historyList *widget.List
}

func NewMainWindow(app fyne.App, cfg *config.Config) *MainWindow {
	// Apply saved language
	if cfg.Language != "" {
		i18n.SetLang(cfg.Language)
	}
	return &MainWindow{
		app:         app,
		cfg:         cfg,
		relayClient: relay.New(cfg.RelayURL),
	}
}

func (mw *MainWindow) Show(cloudflaredBin string) {
	mw.cloudflaredBin = cloudflaredBin
	mw.win = mw.app.NewWindow(i18n.T("app_title"))
	mw.win.Resize(fyne.NewSize(960, 580))

	tabs := container.NewAppTabs(
		container.NewTabItemWithIcon(i18n.T("tab_connect"), theme.ComputerIcon(), mw.buildConnectTab()),
		container.NewTabItemWithIcon(i18n.T("tab_history"), theme.HistoryIcon(), mw.buildHistoryTab()),
		container.NewTabItemWithIcon(i18n.T("tab_settings"), theme.SettingsIcon(), mw.buildSettingsTab()),
	)
	tabs.SetTabLocation(container.TabLocationTop)

	mw.win.SetContent(tabs)
	mw.win.SetOnClosed(func() { mw.stopHost() })

	// Auto-start host sharing immediately
	go func() {
		time.Sleep(300 * time.Millisecond) // let window render first
		mw.startHost()
	}()

	mw.win.ShowAndRun()
}

// ─── Connect Tab ─────────────────────────────────────────────────────────────

func (mw *MainWindow) buildConnectTab() fyne.CanvasObject {
	// HOST PANEL
	mw.hostIDLbl = widget.NewLabel(mw.cfg.HostID)
	mw.hostIDLbl.TextStyle = fyne.TextStyle{Monospace: true, Bold: true}

	copyBtn := widget.NewButtonWithIcon(i18n.T("copy_id"), theme.ContentCopyIcon(), func() {
		mw.win.Clipboard().SetContent(mw.cfg.HostID)
		dialog.ShowInformation(i18n.T("copied"), i18n.T("copied"), mw.win)
	})

	regenBtn := widget.NewButton(i18n.T("regen_id"), func() {
		dialog.ShowConfirm(i18n.T("regen_id"), i18n.T("regen_confirm"),
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
	passEntry.SetPlaceHolder(i18n.T("no_password"))
	passEntry.SetText(mw.cfg.HostPassword)
	passEntry.OnChanged = func(s string) {
		mw.cfg.HostPassword = s
		mw.cfg.Save()
	}

	mw.hostStatusLbl = widget.NewLabel(i18n.T("status_starting"))
	mw.hostStatusLbl.Wrapping = fyne.TextWrapWord

	hostCard := widget.NewCard(i18n.T("your_pc"), "", container.NewVBox(
		container.NewGridWithColumns(2,
			widget.NewLabel(i18n.T("your_id")),
			container.NewHBox(mw.hostIDLbl, copyBtn),
		),
		regenBtn,
		widget.NewSeparator(),
		widget.NewLabel(i18n.T("access_password")),
		passEntry,
		widget.NewSeparator(),
		mw.hostStatusLbl,
	))

	// CLIENT PANEL
	mw.connectIDEntry = widget.NewEntry()
	mw.connectIDEntry.SetPlaceHolder(i18n.T("id_placeholder"))

	mw.connectPassEntry = widget.NewPasswordEntry()
	mw.connectPassEntry.SetPlaceHolder(i18n.T("if_required"))

	mw.rememberChk = widget.NewCheck(i18n.T("remember_password"), nil)

	mw.connectIDEntry.OnChanged = func(s string) {
		if pw, ok := mw.cfg.GetHistoryPassword(s); ok {
			mw.connectPassEntry.SetText(pw)
			mw.rememberChk.SetChecked(true)
		}
	}

	connectBtn := widget.NewButtonWithIcon(i18n.T("connect_btn"), theme.LoginIcon(), func() {
		hostID := mw.connectIDEntry.Text
		if hostID == "" {
			dialog.ShowError(fmt.Errorf("%s", i18n.T("enter_host_id")), mw.win)
			return
		}
		go mw.connectToHost(hostID, mw.connectPassEntry.Text, mw.rememberChk.Checked)
	})
	connectBtn.Importance = widget.HighImportance

	clientCard := widget.NewCard(i18n.T("connect_to_host"), "", container.NewVBox(
		widget.NewLabel(i18n.T("id_host")),
		mw.connectIDEntry,
		widget.NewLabel(i18n.T("password")),
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
				widget.NewLabel(""),
				widget.NewLabel(""),
				widget.NewButton(i18n.T("connect"), nil),
				widget.NewButton(i18n.T("remove"), nil),
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

			row.Objects[2].(*widget.Button).OnTapped = func() {
				pw := entry.Password
				if !entry.RememberPassword {
					pw = ""
				}
				go mw.connectToHost(entry.ID, pw, entry.RememberPassword)
			}
			row.Objects[3].(*widget.Button).OnTapped = func() {
				mw.cfg.RemoveHistory(entry.ID)
				mw.cfg.Save()
				mw.historyList.Refresh()
			}
		},
	)

	emptyNote := widget.NewLabelWithStyle(
		i18n.T("history_empty"),
		fyne.TextAlignCenter,
		fyne.TextStyle{Italic: true},
	)
	return container.NewStack(container.NewPadded(emptyNote), mw.historyList)
}

// ─── Settings Tab ─────────────────────────────────────────────────────────────

func (mw *MainWindow) buildSettingsTab() fyne.CanvasObject {
	// Language selector
	langNames := make([]string, len(i18n.Languages))
	for idx, code := range i18n.Languages {
		langNames[idx] = i18n.LangNames[code]
	}
	langSelect := widget.NewSelect(langNames, nil)
	// Set current
	for idx, code := range i18n.Languages {
		if code == i18n.Lang() {
			langSelect.SetSelectedIndex(idx)
			break
		}
	}

	// Autostart checkbox
	autostartChk := widget.NewCheck(i18n.T("startup_with_os"), nil)
	autostartChk.SetChecked(autostart.IsEnabled())

	// Relay URL
	relayEntry := widget.NewEntry()
	relayEntry.SetText(mw.cfg.RelayURL)

	saveBtn := widget.NewButtonWithIcon(i18n.T("save"), theme.DocumentSaveIcon(), func() {
		// Save language
		if langSelect.SelectedIndex() >= 0 {
			code := i18n.Languages[langSelect.SelectedIndex()]
			i18n.SetLang(code)
			mw.cfg.Language = code
		}

		// Save relay URL
		if relayEntry.Text != "" {
			mw.cfg.RelayURL = relayEntry.Text
			mw.relayClient = relay.New(mw.cfg.RelayURL)
		}

		// Save autostart
		execPath, _ := os.Executable()
		if autostartChk.Checked {
			autostart.Enable(execPath)
		} else {
			autostart.Disable()
		}

		mw.cfg.Save()
		dialog.ShowInformation(i18n.T("saved"), i18n.T("saved"), mw.win)
	})
	saveBtn.Importance = widget.HighImportance

	return container.NewPadded(container.NewVBox(
		widget.NewCard(i18n.T("tab_settings"), "", container.NewVBox(
			widget.NewLabel(i18n.T("language")),
			langSelect,
			widget.NewSeparator(),
			autostartChk,
			widget.NewSeparator(),
			widget.NewLabel(i18n.T("relay_url")),
			relayEntry,
			widget.NewSeparator(),
			saveBtn,
		)),
	))
}

// ─── Host lifecycle ───────────────────────────────────────────────────────────

func (mw *MainWindow) startHost() {
	mw.hostStatusLbl.SetText(i18n.T("status_starting"))

	srv := host.NewServer(mw.cfg.HostPassword)
	srv.OnLog = func(msg string) { mw.hostStatusLbl.SetText(msg) }

	port, err := srv.Start()
	if err != nil {
		mw.hostStatusLbl.SetText(fmt.Sprintf("%s: %v", i18n.T("error"), err))
		return
	}
	mw.hostServer = srv

	tun, err := tunnel.Start(port, mw.cloudflaredBin)
	if err != nil {
		mw.hostStatusLbl.SetText(fmt.Sprintf("Tunnel: %v", err))
		srv.Stop()
		return
	}
	mw.hostTunnel = tun

	if err := mw.relayClient.Register(mw.cfg.HostID, tun.URL, mw.cfg.HostPassword); err != nil {
		mw.hostStatusLbl.SetText(
			i18n.T("status_ready") + " " + mw.cfg.HostID +
				"\n(relay offline — URL diretto: " + tun.URL + ")")
	} else {
		mw.hostStatusLbl.SetText(i18n.T("status_ready") + " " + mw.cfg.HostID)
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
	pd := dialog.NewProgress(i18n.T("connection"), i18n.T("searching")+" "+hostID+"…", mw.win)
	pd.Show()

	pd.SetValue(0.2)
	tunnelURL, err := mw.relayClient.Lookup(hostID)
	if err != nil {
		pd.Hide()
		dialog.ShowError(fmt.Errorf("%s: %v", i18n.T("host_not_found"), err), mw.win)
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
