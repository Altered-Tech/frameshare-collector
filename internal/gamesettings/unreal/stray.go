package unreal

import (
	"github.com/alteredtech/frameshare-collector/internal/gamesettings"
	"github.com/alteredtech/frameshare-collector/internal/library"
)

// strayAppID is Stray's Steam app id (store.steampowered.com/app/1332010).
const strayAppID = "1332010"

// strayConfigFolder is the name Unreal saves Stray's config under; see
// configPath.
const strayConfigFolder = "Stray"

func init() {
	gamesettings.Register(strayParser{})
}

// strayParser reads Stray's GameUserSettings.ini (see unreal.go for the
// Unreal Engine layout it shares with other UE titles).
type strayParser struct{}

func (strayParser) Matches(game library.Game, source library.Source) bool {
	return source == library.SourceSteam && game.AppID == strayAppID
}

func (strayParser) ConfigPath(game library.Game) (string, error) {
	return configPath(strayConfigFolder, game)
}

func (strayParser) Parse(data []byte) (gamesettings.TitleSettings, error) {
	return parseGameUserSettings(data)
}
