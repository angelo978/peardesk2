package relay

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func New(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

type RegisterRequest struct {
	ID        string `json:"id"`
	TunnelURL string `json:"tunnel_url"`
	Secret    string `json:"secret"`
}

type RegisterResponse struct {
	OK bool `json:"ok"`
}

func (c *Client) Register(id, tunnelURL, secret string) error {
	body, _ := json.Marshal(RegisterRequest{ID: id, TunnelURL: tunnelURL, Secret: secret})
	resp, err := c.httpClient.Post(c.baseURL+"/relay/register", "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("relay register failed: %s", string(b))
	}
	return nil
}

type LookupResponse struct {
	TunnelURL string `json:"tunnel_url"`
	Name      string `json:"name,omitempty"`
}

func (c *Client) Lookup(id string) (string, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/relay/lookup/" + id)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return "", fmt.Errorf("host non trovato: %s", id)
	}
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("relay lookup failed: %s", string(b))
	}
	var lr LookupResponse
	if err := json.NewDecoder(resp.Body).Decode(&lr); err != nil {
		return "", err
	}
	return lr.TunnelURL, nil
}

type UnregisterRequest struct {
	ID     string `json:"id"`
	Secret string `json:"secret"`
}

func (c *Client) Unregister(id, secret string) error {
	body, _ := json.Marshal(UnregisterRequest{ID: id, Secret: secret})
	req, err := http.NewRequest(http.MethodDelete, c.baseURL+"/relay/unregister", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}
