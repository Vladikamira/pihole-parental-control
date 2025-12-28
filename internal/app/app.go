package app

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/vladikamira/pihole-parental-control/internal/config"
	"github.com/vladikamira/pihole-parental-control/internal/pihole"
	"github.com/vladikamira/pihole-parental-control/internal/speaker"
	"github.com/vladikamira/pihole-parental-control/internal/telegram"
)

func NewApp() *App {
	cfg := config.NewConfig()
	client := pihole.NewClient(cfg)
	tgClient := telegram.NewClient(cfg)
	speakerClient := speaker.NewClient(cfg)

	stats := DomainStats{
		Domains:     cfg.DomainsToCheck,
		GlobalCount: 0,
	}

	return &App{
		cfg:           cfg,
		client:        client,
		tgClient:      tgClient,
		speakerClient: speakerClient,
		stats:         stats,
	}
}

func (a *App) Run() {

	fmt.Printf("Starting app with config: %v\n", a.cfg)

	for {
		if err := checkDomains(a.client, &a.stats); err != nil {
			fmt.Printf("Failed to check domains: %v\n", err)
			continue
		}

		printStats(&a.stats)

		// Block clients that exceeded the limit
		for _, client := range a.stats.Clients {
			remaining := a.cfg.DaylyWatchingLimit - client.TimeWatchedToday
			if !client.Blocked && !client.NotifiedNearLimit && remaining <= 5*time.Minute && remaining > 0 {
				fmt.Printf("Client %s has less than 5 minutes left. Sending notification...\n", client.IP)
				a.tgClient.SendMessage(fmt.Sprintf("Client %s has less than 5 minutes left (%v)", client.IP, remaining.Round(time.Minute)))
				a.speakerClient.Speak(a.cfg.NearLimitMessage)
				client.NotifiedNearLimit = true
			}

			if !client.Blocked && client.TimeWatchedToday > a.cfg.DaylyWatchingLimit {
				fmt.Printf("Client %s reached limit. Blocking...\n", client.IP)
				err := a.client.BlockDomainsForClient(context.Background(), client.IP, a.cfg.DomainsToCheck)
				if err != nil {
					fmt.Printf("Failed to block client %s: %v\n", client.IP, err)
					a.tgClient.SendMessage(fmt.Sprintf("Failed to block client %s: %v", client.IP, err))
				} else {
					client.Blocked = true
					a.tgClient.SendMessage(fmt.Sprintf("Client %s reached limit %s and is now blocked", client.IP, a.cfg.DaylyWatchingLimit))
					a.speakerClient.Speak(a.cfg.LimitReachedMessage)
				}
			}
		}

		if time.Now().After(midnight()) && time.Now().Before(midnight().Add(a.cfg.CheckInternal)) {

			// Unblock all clients
			for _, client := range a.stats.Clients {
				if client.Blocked {
					fmt.Printf("Client %s is blocked. Unblocking...\n", client.IP)
					err := a.client.UnblockDomainsForClient(context.Background(), client.IP)
					if err != nil {
						fmt.Printf("Failed to unblock client %s: %v\n", client.IP, err)
						a.tgClient.SendMessage(fmt.Sprintf("Failed to unblock client %s: %v", client.IP, err))
					} else {
						client.Blocked = false
					}
				}
			}
			resetStats(&a.stats)
		}
		time.Sleep(a.cfg.CheckInternal)
	}
}

func checkDomains(client *pihole.Client, stats *DomainStats) error {

	for _, domain := range stats.Domains {
		queryStats, err := client.GetDomainStats(context.Background(), domain)
		if err != nil {
			fmt.Printf("get domain stats failed: %v\n", err)
			return fmt.Errorf("get domain stats failed: %v", err)
		}

		stats.GlobalCount += len(queryStats.Queries)

		// Sort queries by time to ensure chronological processing
		sort.Slice(queryStats.Queries, func(i, j int) bool {
			return queryStats.Queries[i].Time < queryStats.Queries[j].Time
		})

		for _, query := range queryStats.Queries {

			// check if client exist
			if !checkIfClientExist(stats, query.Client.IP) {
				stats.Clients = append(stats.Clients, NewClientStats(query.Client.IP))
			}

			// update client stats and register and check time interval
			qTime := time.Unix(int64(query.Time), 0)
			updateClientStats(stats, query.Client.IP, qTime)
		}
	}

	return nil
}

func updateClientStats(stats *DomainStats, ip string, t time.Time) {
	for _, client := range stats.Clients {
		if client.IP == ip {
			// Skip if we've already processed this query or newer ones
			if !t.After(client.LastQueryTime) && !client.LastQueryTime.IsZero() {
				return
			}
			client.LastQueryTime = t
			client.RequestsToday++

			if len(client.WatchIntervals) == 0 {
				client.WatchIntervals = append(client.WatchIntervals, WatchIntervals{
					Start:    t,
					End:      t.Add(5 * time.Minute),
					Requests: 1,
				})
				client.TimeWatchedToday += 5 * time.Minute
				return
			}

			// check if last interval is active
			lastInterval := &client.WatchIntervals[len(client.WatchIntervals)-1]
			if t.After(lastInterval.Start) && t.Before(lastInterval.End) {
				lastInterval.Requests++
			} else {
				// check diff with previous window
				diff := t.Sub(lastInterval.End)
				start := t
				if diff < 5*time.Minute {
					start = lastInterval.End
				}

				client.WatchIntervals = append(client.WatchIntervals, WatchIntervals{
					Start:    start,
					End:      start.Add(5 * time.Minute),
					Requests: 1,
				})
				client.TimeWatchedToday += 5 * time.Minute
			}
		}
	}
}

func NewClientStats(ip string) *Client {
	return &Client{
		RequestsToday:    0,
		IP:               ip,
		TimeWatchedToday: time.Duration(0),
	}
}

func checkIfClientExist(stats *DomainStats, ip string) bool {
	for _, client := range stats.Clients {
		if client.IP == ip {
			return true
		}
	}
	return false
}

func printStats(stats *DomainStats) {
	for _, client := range stats.Clients {
		fmt.Printf("Client %s timeWatched: %v Requests: %d\n", client.IP, client.TimeWatchedToday, client.RequestsToday)
		for _, interval := range client.WatchIntervals {
			fmt.Printf("  Interval Start: %s End: %s Requests: %d\n", interval.Start, interval.End, interval.Requests)
		}
	}
}

func midnight() time.Time {
	return time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.Now().Location())
}

func resetStats(stats *DomainStats) {
	for _, client := range stats.Clients {
		client.RequestsToday = 0
		client.TimeWatchedToday = 0
		client.WatchIntervals = nil
		client.NotifiedNearLimit = false
	}
}
