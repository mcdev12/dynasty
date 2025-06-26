package sport_radar_client

import (
	"fmt"
	"strings"

	"github.com/mcdev12/dynasty/go/clients"
)

type SportRadarClient struct {
	*clients.BaseClient
	apiKey string
}

func NewSportRadarClient(apiKey string) *SportRadarClient {
	client := &SportRadarClient{
		BaseClient: clients.NewBaseClient(BaseURL),
		apiKey:     apiKey,
	}

	client.SetHeader(JsonHeader, JsonContentType)

	return client
}

// Get overrides the base Get method to add API key query parameter
func (c *SportRadarClient) Get(endpoint string) ([]byte, error) {
	// Add API key as query parameter
	separator := "?"
	if strings.Contains(endpoint, "?") {
		separator = "&"
	}
	endpointWithKey := fmt.Sprintf("%s%s%s=%s", endpoint, separator, APIKeyParam, c.apiKey)
	
	return c.BaseClient.Get(endpointWithKey)
}
