# PearDesk

Desktop remoto tramite Cloudflare Tunnel — interfaccia nativa leggera, nessun Electron.

## Funzionalità

- **Connessione tramite Cloudflare Tunnel** — nessun IP locale, nessuna LAN, nessuna porta aperta
- **ID host a 3 blocchi** (`ABC-123-XYZ`) — fisso, rigenerabile on-demand
- **Password con opzione "Ricorda"** — per ogni client, identico a RustDesk
- **Cronologia connessioni** — riconnessione con un clic, password salvata
- **Finestra remota separata** — la GUI principale resta libera
- **Controllo mouse e tastiera** — funziona direttamente nella finestra remota
- **Trasferimento file bidirezionale** — in finestra dedicata
- **Scaling adattivo** — l'immagine scala senza cambiare la risoluzione reale del PC remoto

## Architettura

```
┌──────────────────────────────────────────────────────────┐
│  HOST PC                                                 │
│  PearDesk (host mode)                                    │
│    └── cloudflared quick tunnel → trycloudflare.com URL  │
│    └── WebSocket server (localhost:PORT)                 │
│    └── Screen capture + input injection                  │
└──────────────────────────────────┬───────────────────────┘
                                   │ registra ID → URL
                                   ▼
                          Relay Server (API)
                          /api/relay/register
                          /api/relay/lookup/:id
                                   │ lookup ID → URL
                                   ▼
┌──────────────────────────────────────────────────────────┐
│  CLIENT PC                                               │
│  PearDesk (client mode)                                  │
│    └── inserisce ID → interroga relay → ottiene URL      │
│    └── apre WebSocket a wss://xxx.trycloudflare.com/ws   │
│    └── riceve frame JPEG, invia eventi mouse/tastiera    │
└──────────────────────────────────────────────────────────┘
```

## Compilazione

### Prerequisiti

**Comuni:**
- Go 1.22+

**Linux:**
```bash
sudo apt-get install -y \
  libx11-dev libxrandr-dev libxcursor-dev libxi-dev libxinerama-dev \
  libgl1-mesa-dev libgles2-mesa-dev libfontconfig1-dev libfreetype6-dev \
  libxtst-dev pkg-config gcc
```

**Windows:** MinGW-w64 (`x86_64-w64-mingw32-gcc`)
```bash
# Ubuntu/Debian
sudo apt-get install gcc-mingw-w64-x86-64
```

**macOS:** Xcode Command Line Tools
```bash
xcode-select --install
```

### Cloudflared

PearDesk usa [cloudflared](https://developers.cloudflare.com/cloudflare-one/connections/connect-apps/install-and-setup/installation/) per creare i tunnel. Installalo sul PC host:

```bash
# Linux
sudo apt install cloudflared
# oppure
curl -L --output cloudflared.deb https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-amd64.deb
sudo dpkg -i cloudflared.deb

# macOS
brew install cloudflare/cloudflare/cloudflared

# Windows
winget install Cloudflare.cloudflared
```

PearDesk cerca cloudflared automaticamente nel PATH.

### Build manuale

```bash
# Clone
git clone https://github.com/youruser/peardesk.git
cd peardesk

# Linux
make linux

# AppImage (richiede appimagetool)
make appimage

# Windows (cross-compile da Linux, richiede mingw-w64)
make windows

# macOS (solo da macOS)
make macos

# Zip sorgenti
make package
```

### CI/CD automatica (GitHub Actions)

Il file `.github/workflows/build.yml` compila automaticamente per tutti i sistemi
quando si crea un tag `v*`:

```bash
git tag v1.0.0
git push origin v1.0.0
```

Gli artefatti (AppImage, exe, macOS zip) vengono allegati alla release GitHub.

## Relay Server

Il relay server è l'unico componente "sempre acceso". Mappa gli ID host agli URL dei tunnel Cloudflare. È già incluso nell'API server del progetto.

Endpoint:
- `POST /api/relay/register` — registra un host
- `GET /api/relay/lookup/:id` — ottieni l'URL del tunnel per un ID
- `DELETE /api/relay/unregister` — rimuovi un host
- `GET /api/relay/status` — host attivi

Il relay server incluso in questo progetto è raggiungibile a:
`https://3ca850fa-31e6-4a64-9bf4-cb64713888d9-00-xcum3wq713r4.picard.replit.dev/api`

Per usare il tuo relay server, cambia `relay_url` in `~/.peardesk/config.json`.

## Struttura del codice

```
peardesk/
├── cmd/peardesk/main.go        # Entry point
├── pkg/
│   ├── config/config.go        # Config e cronologia locale (~/.peardesk/)
│   ├── id/id.go               # Generazione ID 3-blocchi alfanumerici
│   ├── protocol/types.go      # Tipi messaggi WebSocket
│   ├── relay/relay.go         # Client REST del relay server
│   ├── tunnel/tunnel.go       # Gestione processo cloudflared
│   ├── host/
│   │   ├── server.go          # Server WebSocket (autenticazione + streaming)
│   │   ├── capture.go         # Screen capture (kbinani/screenshot)
│   │   ├── input.go           # Iniezione mouse/tastiera (robotgo)
│   │   └── files.go           # Trasferimento file
│   ├── client/
│   │   └── connection.go      # Client WebSocket
│   └── ui/
│       ├── main_window.go     # Finestra principale (split host/client)
│       ├── remote_window.go   # Finestra schermo remoto
│       └── file_window.go     # Finestra trasferimento file
├── assets/icon.png            # Icona pera rossa
├── .github/workflows/build.yml # CI per AppImage + exe + macOS
├── Makefile                   # Build locale
└── go.mod
```


