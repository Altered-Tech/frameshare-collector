package source

import (
	"github.com/alteredtech/frameshare-collector/internal/gamesettings"
	"github.com/alteredtech/frameshare-collector/internal/library"
)

// tf2AppID is Team Fortress 2's Steam app id (store.steampowered.com/app/440).
const tf2AppID = "440"

// tf2ModDir is TF2's Source engine "mod" directory, which holds its
// cfg/video.txt; see configPath.
const tf2ModDir = "tf"

func init() {
	gamesettings.Register(teamFortress2Parser{})
}

// teamFortress2Parser reads Team Fortress 2's cfg/video.txt (see
// source.go for the Source engine layout it shares with other titles on
// the same engine).
type teamFortress2Parser struct{}

func (teamFortress2Parser) Matches(game library.Game, source library.Source) bool {
	return source == library.SourceSteam && game.AppID == tf2AppID
}

func (teamFortress2Parser) ConfigPath(game library.Game) (string, error) {
	return configPath(game, tf2ModDir)
}

func (teamFortress2Parser) Parse(data []byte) (gamesettings.TitleSettings, error) {
	return parseVideoTxt(data)
}
