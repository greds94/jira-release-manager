package jira

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/spf13/viper"
)

// Client rappresenta un client per le API Jira
type Client struct {
	BaseURL    string
	Username   string
	APIToken   string
	HTTPClient *http.Client
}

// NewClient crea e restituisce un client Jira configurato.
func NewClient() (*Client, error) {
	jiraURL := viper.GetString("JIRA_URL")
	username := viper.GetString("JIRA_USERNAME")
	apiToken := viper.GetString("JIRA_API_TOKEN")

	if jiraURL == "" || username == "" || apiToken == "" {
		return nil, fmt.Errorf("le credenziali Jira (JIRA_URL, JIRA_USERNAME, JIRA_API_TOKEN) non sono configurate")
	}

	// Rimuovi trailing slash dall'URL se presente
	jiraURL = strings.TrimSuffix(jiraURL, "/")

	return &Client{
		BaseURL:    jiraURL,
		Username:   username,
		APIToken:   apiToken,
		HTTPClient: &http.Client{},
	}, nil
}

// DoRequest esegue una richiesta HTTP con autenticazione
func (c *Client) DoRequest(method, endpoint string, body io.Reader) ([]byte, error) {
	url := c.BaseURL + endpoint

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("errore nella creazione della richiesta: %w", err)
	}

	// Autenticazione Basic Auth
	req.SetBasicAuth(c.Username, c.APIToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("errore nella richiesta HTTP: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("errore nella lettura della risposta: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("errore HTTP %d: %s", resp.StatusCode, string(responseBody))
	}

	return responseBody, nil
}

// GetJSON esegue una richiesta GET e decodifica il JSON
func (c *Client) GetJSON(endpoint string, v interface{}) error {
	data, err := c.DoRequest("GET", endpoint, nil)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("errore nel parsing JSON: %w", err)
	}

	return nil
}
