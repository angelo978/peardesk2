package config

import (
        "encoding/json"
        "os"
        "path/filepath"
        "time"
)

type HistoryEntry struct {
        ID              string    `json:"id"`
        Name            string    `json:"name"`
        LastConnected   time.Time `json:"last_connected"`
        RememberPassword bool     `json:"remember_password"`
        Password        string    `json:"password,omitempty"`
}

type Config struct {
        HostID       string         `json:"host_id"`
        HostPassword string         `json:"host_password"`
        History      []HistoryEntry `json:"history"`
        RelayURL     string         `json:"relay_url"`
        Language     string         `json:"language,omitempty"`
}

var defaultRelayURL = "https://3ca850fa-31e6-4a64-9bf4-cb64713888d9-00-xcum3wq713r4.picard.replit.dev/api"

func dataDir() string {
        home, err := os.UserHomeDir()
        if err != nil {
                return ".peardesk"
        }
        return filepath.Join(home, ".peardesk")
}

func configPath() string {
        return filepath.Join(dataDir(), "config.json")
}

func Load() (*Config, error) {
        _ = os.MkdirAll(dataDir(), 0700)
        data, err := os.ReadFile(configPath())
        if err != nil {
                if os.IsNotExist(err) {
                        return &Config{RelayURL: defaultRelayURL}, nil
                }
                return nil, err
        }
        var cfg Config
        if err := json.Unmarshal(data, &cfg); err != nil {
                return &Config{RelayURL: defaultRelayURL}, nil
        }
        if cfg.RelayURL == "" {
                cfg.RelayURL = defaultRelayURL
        }
        return &cfg, nil
}

func (c *Config) Save() error {
        _ = os.MkdirAll(dataDir(), 0700)
        data, err := json.MarshalIndent(c, "", "  ")
        if err != nil {
                return err
        }
        return os.WriteFile(configPath(), data, 0600)
}

func (c *Config) AddOrUpdateHistory(id, name, password string, remember bool) {
        for i, h := range c.History {
                if h.ID == id {
                        c.History[i].Name = name
                        c.History[i].LastConnected = time.Now()
                        c.History[i].RememberPassword = remember
                        if remember {
                                c.History[i].Password = password
                        } else {
                                c.History[i].Password = ""
                        }
                        return
                }
        }
        entry := HistoryEntry{
                ID:              id,
                Name:            name,
                LastConnected:   time.Now(),
                RememberPassword: remember,
        }
        if remember {
                entry.Password = password
        }
        c.History = append([]HistoryEntry{entry}, c.History...)
        if len(c.History) > 50 {
                c.History = c.History[:50]
        }
}

func (c *Config) GetHistoryPassword(id string) (string, bool) {
        for _, h := range c.History {
                if h.ID == id && h.RememberPassword {
                        return h.Password, true
                }
        }
        return "", false
}

func (c *Config) RemoveHistory(id string) {
        filtered := c.History[:0]
        for _, h := range c.History {
                if h.ID != id {
                        filtered = append(filtered, h)
                }
        }
        c.History = filtered
}

func CloudflaredPath() string {
        return filepath.Join(dataDir(), "cloudflared")
}
