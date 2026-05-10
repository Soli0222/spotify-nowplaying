package handler

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidatePublicHTTPSURL_PublicIP(t *testing.T) {
	result, err := validatePublicHTTPSURL("https://8.8.8.8/")

	require.NoError(t, err)
	assert.Equal(t, "https://8.8.8.8", result)
}

func TestValidatePublicHTTPSURL_RejectsUnsafeURLs(t *testing.T) {
	tests := []string{
		"http://8.8.8.8",
		"https://127.0.0.1",
		"https://10.0.0.1",
		"https://172.16.0.1",
		"https://192.168.0.1",
		"https://169.254.169.254",
		"https://user@example.com",
	}

	for _, rawURL := range tests {
		t.Run(rawURL, func(t *testing.T) {
			_, err := validatePublicHTTPSURL(rawURL)
			assert.Error(t, err)
		})
	}
}
