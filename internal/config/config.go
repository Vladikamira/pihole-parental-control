package config

import (
	"os"
	"time"
)

type Config struct {
	PiholeAddress       string
	CheckInternal       time.Duration
	DaylyWatchingLimit  time.Duration
	DomainsToCheck      []string
	Password            string
	TelegramToken       string
	TelegramChatID      string
	SpeakerURL          string
	SpeakerLanguage     string
	NearLimitMessage    string
	LimitReachedMessage string
	ApiPort             string
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
		Password:            os.Getenv("PIHOLE_PASSWORD"),
		TelegramToken:       os.Getenv("TELEGRAM_BOT_TOKEN"),
		TelegramChatID:      os.Getenv("TELEGRAM_CHAT_ID"),
		SpeakerURL:          os.Getenv("SPEAKER_URL"), // e.g. http://192.168.1.50:8080
		SpeakerLanguage:     getEnv("SPEAKER_LANGUAGE", "en"),
		NearLimitMessage:    getEnv("SPEAKER_NEAR_LIMIT_MESSAGE", "Less than five minutes remaining. Wrap it up."),
		LimitReachedMessage: getEnv("SPEAKER_LIMIT_REACHED_MESSAGE", "Time's up. Viewing is now blocked."),
		ApiPort:             getEnv("API_PORT", "8081"),
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
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
