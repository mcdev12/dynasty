package sport_radar_client

const (
	// Base URL - SportRadar uses trial access level by default
	BaseURL = "https://api.sportradar.com/nfl/official/trial/"

	// Paths
	accessLevelTrial    = "trial"
	languageCodeEnglish = "en"

	// Headers - SportRadar uses api_key query parameter, not header
	APIKeyParam     = "api_key"
	JsonHeader      = "accept"
	JsonContentType = "application/json"
)
