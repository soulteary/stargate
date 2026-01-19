// Package warden provides client functionality for interacting with Warden API.
package warden

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/soulteary/stargate/src/internal/config"
)

// AllowListUser represents a user in the allow list.
type AllowListUser struct {
	Phone string `json:"phone"`
	Mail  string `json:"mail"`
}

// Client is the Warden API client.
type Client struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string
	cache      *Cache
	logger     *logrus.Logger
}

// NewClient creates a new Warden API client.
func NewClient() *Client {
	if !config.WardenEnabled.ToBool() {
		return nil
	}

	wardenURL := config.WardenURL.String()
	if wardenURL == "" {
		logrus.Warn("WARDEN_URL is not set, Warden client will not be initialized")
		return nil
	}

	// Remove trailing slash
	wardenURL = strings.TrimSuffix(wardenURL, "/")

	client := &Client{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		baseURL: wardenURL,
		apiKey:  config.WardenAPIKey.String(),
		cache:    NewCache(),
		logger:   logrus.StandardLogger(),
	}

	return client
}

// GetUsers fetches the user list from Warden API.
func (c *Client) GetUsers(ctx context.Context) ([]AllowListUser, error) {
	if c == nil {
		return nil, fmt.Errorf("warden client is not initialized")
	}

	// Check cache first
	if users := c.cache.Get(); users != nil {
		c.logger.Debug("Using cached user list from Warden")
		return users, nil
	}

	// Build request URL
	url := fmt.Sprintf("%s/", c.baseURL)

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add API key header if configured
	if c.apiKey != "" {
		req.Header.Set("X-API-Key", c.apiKey)
		// Also support Authorization header
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	}

	// Make request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch users from Warden: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("warden API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var users []AllowListUser
	if err := json.NewDecoder(resp.Body).Decode(&users); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Update cache
	c.cache.Set(users)

	c.logger.Debugf("Fetched %d users from Warden API", len(users))

	return users, nil
}

// CheckUserInList checks if a user (by phone or mail) is in the allow list.
// Returns false if the user is not found, or if there's an error fetching the user list.
func (c *Client) CheckUserInList(ctx context.Context, phone, mail string) bool {
	if c == nil {
		return false
	}

	users, err := c.GetUsers(ctx)
	if err != nil {
		c.logger.Warnf("Failed to get users from Warden API: %v", err)
		// Return false on error - this allows fallback to password authentication
		return false
	}

	// Normalize input
	phone = strings.TrimSpace(phone)
	mail = strings.TrimSpace(strings.ToLower(mail))

	// Check if user exists
	for _, user := range users {
		userPhone := strings.TrimSpace(user.Phone)
		userMail := strings.TrimSpace(strings.ToLower(user.Mail))

		// Match by phone if provided
		if phone != "" && userPhone == phone {
			return true
		}

		// Match by mail if provided
		if mail != "" && userMail == mail {
			return true
		}
	}

	return false
}

// IsEnabled returns whether Warden integration is enabled.
func IsEnabled() bool {
	return config.WardenEnabled.ToBool()
}
