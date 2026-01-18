package handler

import (
	"net/url"
	"os"
	"strings"
)

// TwitterConfig holds Twitter integration settings
type TwitterConfig struct {
	Enabled        bool
	RequireMisskey bool
	AllowedHosts   []string
	ClientID       string
	ClientSecret   string
}

// LoadTwitterConfig loads Twitter config from environment variables
func LoadTwitterConfig() TwitterConfig {
	config := TwitterConfig{
		Enabled:        true, // default enabled
		RequireMisskey: false,
		AllowedHosts:   nil,
		ClientID:       os.Getenv("TWITTER_CLIENT_ID"),
		ClientSecret:   os.Getenv("TWITTER_CLIENT_SECRET"),
	}

	// TWITTER_ENABLED (default: true)
	if val := os.Getenv("TWITTER_ENABLED"); val == "false" {
		config.Enabled = false
	}

	// TWITTER_REQUIRE_MISSKEY (default: false)
	if val := os.Getenv("TWITTER_REQUIRE_MISSKEY"); val == "true" {
		config.RequireMisskey = true
	}

	// TWITTER_ALLOWED_HOSTS (comma-separated, empty = all allowed)
	if val := os.Getenv("TWITTER_ALLOWED_HOSTS"); val != "" {
		hosts := strings.Split(val, ",")
		config.AllowedHosts = make([]string, 0, len(hosts))
		for _, h := range hosts {
			h = strings.TrimSpace(h)
			if h != "" {
				config.AllowedHosts = append(config.AllowedHosts, strings.ToLower(h))
			}
		}
	}

	return config
}

// IsAvailable returns true if Twitter integration is available (has credentials)
func (c TwitterConfig) IsAvailable() bool {
	return c.Enabled && c.ClientID != "" && c.ClientSecret != ""
}

// TwitterEligibility represents whether a user is eligible for Twitter
type TwitterEligibility struct {
	Eligible bool   `json:"eligible"`
	Reason   string `json:"reason,omitempty"`
}

// CheckEligibility checks if a user is eligible for Twitter integration
// misskeyConnected: whether the user has Misskey connected
// misskeyHost: the user's Misskey instance host (e.g., "misskey.tld")
func (c TwitterConfig) CheckEligibility(misskeyConnected bool, misskeyHost string) TwitterEligibility {
	if !c.Enabled {
		return TwitterEligibility{Eligible: false, Reason: "Twitter integration is disabled"}
	}

	if c.ClientID == "" || c.ClientSecret == "" {
		return TwitterEligibility{Eligible: false, Reason: "Twitter API credentials not configured"}
	}

	if c.RequireMisskey && !misskeyConnected {
		return TwitterEligibility{Eligible: false, Reason: "Misskey connection required"}
	}

	if c.RequireMisskey && len(c.AllowedHosts) > 0 {
		// Extract host from URL (e.g., "https://mi.soli0222.com" -> "mi.soli0222.com")
		host := strings.ToLower(strings.TrimSpace(misskeyHost))
		if parsed, err := url.Parse(host); err == nil && parsed.Host != "" {
			host = parsed.Host
		}
		// Also handle cases where there's no scheme
		host = strings.TrimPrefix(host, "https://")
		host = strings.TrimPrefix(host, "http://")
		host = strings.TrimSuffix(host, "/")

		allowed := false
		for _, h := range c.AllowedHosts {
			if host == h {
				allowed = true
				break
			}
		}
		if !allowed {
			return TwitterEligibility{Eligible: false, Reason: "Your Misskey instance is not in the allowed list"}
		}
	}

	return TwitterEligibility{Eligible: true}
}
