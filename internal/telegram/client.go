package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/vladikamira/pihole-parental-control/internal/config"
)

type Client struct {
	config config.Config
	client *http.Client
}

func NewClient(cfg config.Config) *Client {
	return &Client{
		config: cfg,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *Client) SendMessage(message string) error {
	if c.config.TelegramToken == "" || c.config.TelegramChatID == "" {
		return fmt.Errorf("telegram token or chat id is empty")
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", c.config.TelegramToken)

	payload := map[string]interface{}{
		"chat_id": c.config.TelegramChatID,
		"text":    message,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("send message failed with status %d and body: %s", resp.StatusCode, resp.Body)
	}

	return nil
}
