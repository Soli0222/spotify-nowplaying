package handler

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadTwitterConfig_Defaults(t *testing.T) {
	// Clear any existing environment variables
	t.Setenv("TWITTER_ENABLED", "")
	t.Setenv("TWITTER_REQUIRE_MISSKEY", "")
	t.Setenv("TWITTER_ALLOWED_HOSTS", "")
	t.Setenv("TWITTER_CLIENT_ID", "")
	t.Setenv("TWITTER_CLIENT_SECRET", "")

	config := LoadTwitterConfig()

	assert.True(t, config.Enabled)
	assert.False(t, config.RequireMisskey)
	assert.Nil(t, config.AllowedHosts)
}

func TestLoadTwitterConfig_WithEnv(t *testing.T) {
	t.Setenv("TWITTER_ENABLED", "false")
	t.Setenv("TWITTER_REQUIRE_MISSKEY", "true")
	t.Setenv("TWITTER_ALLOWED_HOSTS", "misskey.tld, mi.example.com")
	t.Setenv("TWITTER_CLIENT_ID", "test-client-id")
	t.Setenv("TWITTER_CLIENT_SECRET", "test-secret")

	config := LoadTwitterConfig()

	assert.False(t, config.Enabled)
	assert.True(t, config.RequireMisskey)
	assert.Len(t, config.AllowedHosts, 2)
	assert.Contains(t, config.AllowedHosts, "misskey.tld")
	assert.Contains(t, config.AllowedHosts, "mi.example.com")
	assert.Equal(t, "test-client-id", config.ClientID)
	assert.Equal(t, "test-secret", config.ClientSecret)
}

func TestTwitterConfig_IsAvailable(t *testing.T) {
	tests := []struct {
		name     string
		config   TwitterConfig
		expected bool
	}{
		{
			name:     "all set",
			config:   TwitterConfig{Enabled: true, ClientID: "id", ClientSecret: "secret"},
			expected: true,
		},
		{
			name:     "disabled",
			config:   TwitterConfig{Enabled: false, ClientID: "id", ClientSecret: "secret"},
			expected: false,
		},
		{
			name:     "no client id",
			config:   TwitterConfig{Enabled: true, ClientID: "", ClientSecret: "secret"},
			expected: false,
		},
		{
			name:     "no client secret",
			config:   TwitterConfig{Enabled: true, ClientID: "id", ClientSecret: ""},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.config.IsAvailable())
		})
	}
}

func TestTwitterConfig_CheckEligibility_Disabled(t *testing.T) {
	config := TwitterConfig{Enabled: false}

	result := config.CheckEligibility(true, "misskey.tld")

	assert.False(t, result.Eligible)
	assert.Contains(t, result.Reason, "disabled")
}

func TestTwitterConfig_CheckEligibility_NoCredentials(t *testing.T) {
	config := TwitterConfig{Enabled: true, ClientID: "", ClientSecret: ""}

	result := config.CheckEligibility(true, "misskey.tld")

	assert.False(t, result.Eligible)
	assert.Contains(t, result.Reason, "credentials")
}

func TestTwitterConfig_CheckEligibility_RequireMisskey(t *testing.T) {
	config := TwitterConfig{
		Enabled:        true,
		ClientID:       "id",
		ClientSecret:   "secret",
		RequireMisskey: true,
	}

	// Not connected
	result := config.CheckEligibility(false, "")
	assert.False(t, result.Eligible)
	assert.Contains(t, result.Reason, "Misskey connection required")

	// Connected
	result = config.CheckEligibility(true, "misskey.tld")
	assert.True(t, result.Eligible)
}

func TestTwitterConfig_CheckEligibility_AllowedHosts(t *testing.T) {
	config := TwitterConfig{
		Enabled:        true,
		ClientID:       "id",
		ClientSecret:   "secret",
		RequireMisskey: true,
		AllowedHosts:   []string{"misskey.tld", "mi.soli0222.com"},
	}

	tests := []struct {
		name             string
		misskeyHost      string
		expectedEligible bool
	}{
		{"exact match", "misskey.tld", true},
		{"exact match 2", "mi.soli0222.com", true},
		{"with https", "https://misskey.tld", true},
		{"with http", "http://misskey.tld", true},
		{"with trailing slash", "misskey.tld/", true},
		{"not allowed", "other.instance.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := config.CheckEligibility(true, tt.misskeyHost)
			assert.Equal(t, tt.expectedEligible, result.Eligible)
		})
	}
}

func TestTwitterConfig_CheckEligibility_NoRequireMisskey(t *testing.T) {
	config := TwitterConfig{
		Enabled:        true,
		ClientID:       "id",
		ClientSecret:   "secret",
		RequireMisskey: false,
	}

	result := config.CheckEligibility(false, "")
	assert.True(t, result.Eligible)
}

func TestTwitterEligibility_JSONTags(t *testing.T) {
	elig := TwitterEligibility{
		Eligible: true,
		Reason:   "test reason",
	}

	assert.True(t, elig.Eligible)
	assert.Equal(t, "test reason", elig.Reason)
}
