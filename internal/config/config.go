package config

import (
	"os"
	"time"
)

type Config struct {
	PiholeAddress      string
	CheckInternal      time.Duration
	DaylyWatchingLimit time.Duration
	DomainsToCheck     []string
	Password           string
	TelegramToken      string
	TelegramChatID     string
}

func NewConfig() Config {
	return Config{
		PiholeAddress:      os.Getenv("PIHOLE_ADDRESS"),
		CheckInternal:      parseDurationEnv("CHECK_INTERNAL", 1*time.Minute),
		DaylyWatchingLimit: parseDurationEnv("DAYLY_WATCHING_LIMIT", 1*time.Hour),
		DomainsToCheck: []string{
			"*youtube*",
			"*googlevideo*",
			"*.ggpht.com",
			"*.youtu.be",
			"*.yt.be",
			"*.ytimg.com",
			"*googleusercontent.com",
		},
		Password:       os.Getenv("PIHOLE_PASSWORD"),
		TelegramToken:  os.Getenv("TELEGRAM_BOT_TOKEN"),
		TelegramChatID: os.Getenv("TELEGRAM_CHAT_ID"),
	}
}

func parseDurationEnv(key string, defaultDuration time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return defaultDuration
	}
	duration, err := time.ParseDuration(value)
	if err != nil {
		return defaultDuration
	}
	return duration
}
