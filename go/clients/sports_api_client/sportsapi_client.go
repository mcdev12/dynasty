package sports_api_client

import (
	"github.com/mcdev12/dynasty/go/clients"
)

type SportsApiClient struct {
	*clients.BaseClient
}

func NewSportsApiClient(apiKey string) *SportsApiClient {
	client := &SportsApiClient{
		BaseClient: clients.NewBaseClient(BaseURL),
	}

	client.SetHeader(RapidAPIKeyHeader, apiKey)
	client.SetHeader(RapidAPIHostHeader, RapidAPIHost)

	return client
}
