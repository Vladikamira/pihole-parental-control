package speaker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/vladikamira/pihole-parental-control/internal/config"
)

type Client struct {
	cfg config.Config
}

func NewClient(cfg config.Config) *Client {
	return &Client{cfg: cfg}
}

type SpeakRequest struct {
	Message  string `json:"message"`
	Language string `json:"language"`
}

func (c *Client) Speak(message string) error {
	if c.cfg.SpeakerURL == "" {
		return nil // Speaker not configured
	}

	reqBody := SpeakRequest{
		Message:  message,
		Language: c.cfg.SpeakerLanguage,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal speak request: %v", err)
	}

	url := fmt.Sprintf("%s/speak", c.cfg.SpeakerURL)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send speak request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("speaker service returned status: %s", resp.Status)
	}

	return nil
}
