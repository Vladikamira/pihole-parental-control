package app

import (
	"time"

	"github.com/vladikamira/pihole-parental-control/internal/config"
	"github.com/vladikamira/pihole-parental-control/internal/pihole"
	"github.com/vladikamira/pihole-parental-control/internal/telegram"
)

type WatchIntervals struct {
	Start    time.Time
	End      time.Time
	Requests int
}

type App struct {
	cfg      config.Config
	client   *pihole.Client
	tgClient *telegram.Client
	stats    DomainStats
}

type Client struct {
	RequestsToday     int
	IP                string
	TimeWatchedToday  time.Duration
	WatchIntervals    []WatchIntervals
	LastQueryTime     time.Time
	Blocked           bool
	NotifiedNearLimit bool
}

type DomainStats struct {
	Domains     []string `json:"domain"`
	GlobalCount int      `json:"count"`
	Clients     []*Client
}
