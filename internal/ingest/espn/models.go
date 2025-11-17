package espn

import (
	"time"

	"github.com/fortuna/minerva/internal/store"
)

// TeamMeta captures ESPN team identifiers needed for mapping.
type TeamMeta struct {
	Abbreviation string
	ESPNID       string
	DisplayName  string
}

// ParsedGame wraps a store.Game together with source team metadata.
type ParsedGame struct {
	Game      *store.Game
	HomeTeam  TeamMeta
	AwayTeam  TeamMeta
	SeasonType string
}

// ParsedPlayerStats bundles player metadata with box score output.
type ParsedPlayerStats struct {
	Stats         *store.PlayerGameStats
	TeamAbbr      string
	ESPNPlayerID  string
	PlayerName    string
	Position      string
	Jersey        string
	Height        string
	Weight        int
	BirthDate     *time.Time
}


