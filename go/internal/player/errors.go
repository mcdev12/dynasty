package player

import "errors"

// ErrNoProfile is returned when a player has no sport-specific profile
var ErrNoProfile = errors.New("no profile")