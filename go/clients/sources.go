package clients

// ExternalSource represents different external data providers
type ExternalSource string

const (
	// ExternalSourceSportsAPI represents the current sports API client
	ExternalSourceSportsAPI ExternalSource = "sportsapi"

	// ExternalSourceSportsDataIO represents sportsdataio API
	ExternalSourceSportsDataIO ExternalSource = "sportsdataio"

	// ExternalSourceESPN represents ESPN API
	ExternalSourceESPN ExternalSource = "espn"

	// ExternalSourceManual represents manually entered data
	ExternalSourceManual ExternalSource = "manual"
)

// ExternalSourceConfig holds configuration for external sources
type ExternalSourceConfig struct {
	Source      ExternalSource `json:"source"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Priority    int            `json:"priority"` // Higher priority sources override lower ones
	Active      bool           `json:"active"`
}

// GetExternalSources returns all configured external sources
func GetExternalSources() map[ExternalSource]ExternalSourceConfig {
	return map[ExternalSource]ExternalSourceConfig{
		ExternalSourceSportsAPI: {
			Source:      ExternalSourceSportsAPI,
			Name:        "Sports API",
			Description: "Current sports API client",
			Priority:    100,
			Active:      true,
		},
		ExternalSourceSportsDataIO: {
			Source:      ExternalSourceSportsDataIO,
			Name:        "SportsData.io",
			Description: "SportsData.io API provider",
			Priority:    90,
			Active:      false,
		},
		ExternalSourceESPN: {
			Source:      ExternalSourceESPN,
			Name:        "ESPN API",
			Description: "ESPN sports data API",
			Priority:    80,
			Active:      false,
		},
		ExternalSourceManual: {
			Source:      ExternalSourceManual,
			Name:        "Manual Entry",
			Description: "Manually entered team data",
			Priority:    10,
			Active:      true,
		},
	}
}

// ValidateExternalSource checks if the source is valid
func ValidateExternalSource(source ExternalSource) bool {
	sources := GetExternalSources()
	_, exists := sources[source]
	return exists
}

// GetActiveExternalSources returns only active external sources
func GetActiveExternalSources() map[ExternalSource]ExternalSourceConfig {
	all := GetExternalSources()
	active := make(map[ExternalSource]ExternalSourceConfig)

	for source, config := range all {
		if config.Active {
			active[source] = config
		}
	}

	return active
}

// GetHighestPrioritySource returns the external source with highest priority
func GetHighestPrioritySource() ExternalSource {
	sources := GetActiveExternalSources()
	var highest ExternalSource
	var highestPriority int

	for source, config := range sources {
		if config.Priority > highestPriority {
			highest = source
			highestPriority = config.Priority
		}
	}

	return highest
}
