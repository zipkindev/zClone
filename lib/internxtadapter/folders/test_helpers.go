package folders

import (
	"zclone/lib/internxtadapter/config"
	"zclone/lib/internxtadapter/endpoints"
)

// newTestConfig creates a test config with the given mock server URL.
// The HTTPClient is properly configured with the centralized header transport.
func newTestConfig(mockServerURL string) *config.Config {
	cfg := &config.Config{
		Token:     "test-token",
		Endpoints: endpoints.NewConfig(mockServerURL),
	}
	cfg.ApplyDefaults()
	return cfg
}
