package unreal

import (
	"github.com/alteredtech/frameshare-collector/internal/gamesettings"
	"github.com/alteredtech/frameshare-collector/internal/library"
)

// strayAppID is Stray's Steam app id (store.steampowered.com/app/1332010).
const strayAppID = "1332010"

// strayConfigFolder is the name Unreal saves Stray's config under; see
// configPath. Stray's own project name in Unreal is "Hk_project", not
// "Stray" -- config lands under that folder, not one named after the
// title.
const strayConfigFolder = "Hk_project"

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
