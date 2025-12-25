package pihole

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/vladikamira/pihole-parental-control/internal/config"
)

func NewClient(config config.Config) *Client {
	return &Client{
		config: config,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

type Client struct {
	config            config.Config
	client            *http.Client
	sessionID         string
	sessionExpiration time.Time
}

func (c *Client) Auth(ctx context.Context) error {
	if c.sessionID != "" && time.Now().Before(c.sessionExpiration) {
		return nil
	}

	data, err := json.Marshal(map[string]string{"password": c.config.Password})
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", c.config.PiholeAddress+"/api/auth", bytes.NewBuffer(data))
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
		return fmt.Errorf("request failed: %s", resp.Status)
	}

	var authResponse AuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResponse); err != nil {
		return err
	}

	if !authResponse.Session.Valid {
		return fmt.Errorf("session is not valid: %s", authResponse.Session.Message)
	}

	c.sessionID = authResponse.Session.Sid
	c.sessionExpiration = time.Now().Add(time.Duration(authResponse.Session.Validity) * time.Second)

	fmt.Println("Auth successeded. Got session id: ", c.sessionID)

	return nil
}

func (c *Client) GetDomainStats(ctx context.Context, domain string) (*QueryStats, error) {
	if err := c.Auth(ctx); err != nil {
		return nil, err
	}

	from := strconv.FormatInt(time.Now().Add(-c.config.CheckInternal).Unix(), 10)
	until := strconv.FormatInt(time.Now().Unix(), 10)

	req, err := http.NewRequest("GET", c.config.PiholeAddress+"/api/queries?domain="+domain+"&from="+from+"&until="+until, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-FTL-SID", c.sessionID)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, err
	}
	var stats QueryStats
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return nil, err
	}
	return &stats, nil
}

func (c *Client) BlockDomainsForClient(ctx context.Context, clientIP string, domains []string) error {
	if err := c.Auth(ctx); err != nil {
		return err
	}

	groupName := fmt.Sprintf("ParentalControl-%s", clientIP)
	groupID, err := c.getOrCreateGroup(ctx, groupName)
	if err != nil {
		return fmt.Errorf("failed to get/create group: %w", err)
	}

	fmt.Printf("Group %s created with id: %d\n", groupName, groupID)

	// Add domains to group
	for _, domain := range domains {
		fmt.Printf("Adding domain %s to group: %d\n", domain, groupID)
		if err := c.addDomainToGroup(ctx, domain, groupID); err != nil {
			return fmt.Errorf("failed to add domain %s: %w", domain, err)
		}
	}

	// Add client to group
	if err := c.addClientToGroup(ctx, clientIP, groupID); err != nil {
		return fmt.Errorf("failed to add client to group: %w", err)
	}

	return nil
}

func (c *Client) UnblockDomainsForClient(ctx context.Context, clientIP string) error {
	if err := c.Auth(ctx); err != nil {
		return err
	}

	groupName := fmt.Sprintf("ParentalControl-%s", clientIP)
	groupID, err := c.getGroupID(ctx, groupName)
	if err != nil {
		return fmt.Errorf("failed to get group id: %w", err)
	}

	// Remove client from group
	if err := c.removeClientFromGroup(ctx, clientIP, groupID); err != nil {
		return fmt.Errorf("failed to remove client from group: %w", err)
	}

	return nil
}

// Helpers

func (c *Client) getOrCreateGroup(ctx context.Context, name string) (int, error) {
	id, err := c.getGroupID(ctx, name)
	if err == nil {
		return id, nil
	}

	// Create group
	data, _ := json.Marshal(map[string]string{
		"name":        name,
		"description": "Created by Parental Control",
	})
	req, _ := http.NewRequest("POST", c.config.PiholeAddress+"/api/groups", bytes.NewBuffer(data))
	req.Header.Set("X-FTL-SID", c.sessionID)
	req.Header.Set("Content-Type", "application/json") // Ensure Content-Type is set

	resp, err := c.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return 0, fmt.Errorf("failed to create group, status: %d", resp.StatusCode)
	}

	// Assume response contains ID? Or we fetch again.
	// Pi-hole v6 API usually returns the object.
	var res GroupResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		// Fallback: fetch by name
		return c.getGroupID(ctx, name)
	}
	fmt.Println("Group created with id: ", res.Groups[0].ID)
	return res.Groups[0].ID, nil
}

func (c *Client) getGroupID(ctx context.Context, name string) (int, error) {
	req, _ := http.NewRequest("GET", c.config.PiholeAddress+"/api/groups", nil)
	req.Header.Set("X-FTL-SID", c.sessionID)
	resp, err := c.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var groups GroupListResponse
	// Note: Response structure might differ, simplified assumption
	if err := json.NewDecoder(resp.Body).Decode(&groups); err != nil {
		return 0, err
	}

	for _, g := range groups.Groups {
		if g.Name == name {
			return g.ID, nil
		}
	}
	return 0, fmt.Errorf("group not found")
}

func (c *Client) addDomainToGroup(ctx context.Context, domain string, groupID int) error {
	// Add domain (blacklist regex or exact)
	payload := map[string]interface{}{
		"domain":  domain,
		"comment": "Parental Control",
		"groups":  []int{groupID},
		"enabled": true,
	}
	data, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", c.config.PiholeAddress+"/api/domains/deny/regex", bytes.NewBuffer(data))
	req.Header.Set("X-FTL-SID", c.sessionID)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("add domain failed: %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (c *Client) addClientToGroup(ctx context.Context, ip string, groupID int) error {
	// First get client to preserve existing groups
	client, err := c.getClient(ctx, ip)
	exists := true
	if err != nil {
		// Client might not exist in database, create query
		// But in Pi-hole clients are identified by IP.
		// If we POST a new client, we specify groups.
		client = &ClientItem{IP: ip, Groups: []int{}}
		exists = false
	}

	// Check if already in group
	for _, gid := range client.Groups {
		if gid == groupID {
			return nil
		}
	}
	client.Groups = append(client.Groups, groupID)

	if !exists {
		return c.createClient(ctx, client)
	}

	return c.updateClient(ctx, client)
}

func (c *Client) removeClientFromGroup(ctx context.Context, ip string, groupID int) error {
	client, err := c.getClient(ctx, ip)
	if err != nil {
		return fmt.Errorf("failed to get client: %w", err)
	}

	newGroups := []int{}
	found := false
	for _, gid := range client.Groups {
		if gid == groupID {
			found = true
			continue
		}
		newGroups = append(newGroups, gid)
	}

	if !found {
		return fmt.Errorf("client %s not found in group %d", ip, groupID)
	}
	client.Groups = newGroups
	return c.updateClient(ctx, client)
}

func (c *Client) getClient(ctx context.Context, ip string) (*ClientItem, error) {
	// Need to list clients or get specific?
	// Assuming GET /api/clients/{ip} or search
	req, _ := http.NewRequest("GET", c.config.PiholeAddress+"/api/clients", nil) // List all?
	req.Header.Set("X-FTL-SID", c.sessionID)
	resp, err := c.client.Do(req)
	if err != nil {
		fmt.Printf("getClient network error: %v\n", err)
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	// fmt.Printf("DEBUG: getClient response: %s\n", string(bodyBytes))

	var response struct {
		Clients []ClientItem `json:"clients"`
	}
	// Try parsing as wrapped object
	if err := json.Unmarshal(bodyBytes, &response); err != nil {
		// Try parsing as array directly?
		// Some versions might return just []ClientItem
		var clients []ClientItem
		if err2 := json.Unmarshal(bodyBytes, &clients); err2 == nil {
			response.Clients = clients
		} else {
			fmt.Printf("getClient decode error: %v\nBody: %s\n", err, string(bodyBytes))
			return nil, err
		}
	}

	for _, cl := range response.Clients {
		if cl.IP == ip {
			return &cl, nil
		}
	}
	fmt.Printf("getClient: IP %s not found in %d clients\n", ip, len(response.Clients))
	return nil, fmt.Errorf("client not found")
}

func (c *Client) createClient(ctx context.Context, client *ClientItem) error {
	data, _ := json.Marshal(client)
	req, _ := http.NewRequest("POST", c.config.PiholeAddress+"/api/clients", bytes.NewBuffer(data))
	req.Header.Set("X-FTL-SID", c.sessionID)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("create client failed: %d, body: %s", resp.StatusCode, string(body))
	}
	return nil
}

func (c *Client) updateClient(ctx context.Context, client *ClientItem) error {
	data, _ := json.Marshal(client)
	// Use PUT and append IP to URL for update
	req, _ := http.NewRequest("PUT", c.config.PiholeAddress+"/api/clients/"+client.IP, bytes.NewBuffer(data))
	req.Header.Set("X-FTL-SID", c.sessionID)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("update client failed: %d, body: %s", resp.StatusCode, string(body))
	}
	return nil
}
