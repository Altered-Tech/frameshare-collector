// Package all registers every supported title's gamesettings.Parser.
// Blank-import it (as opposed to the individual engine/format
// subpackages) when you want gamesettings.Collect to be able to find any
// title this repo supports, e.g. from cmd/collector:
//
//	import _ "github.com/alteredtech/frameshare-collector/internal/gamesettings/all"
package all

import (
	_ "github.com/alteredtech/frameshare-collector/internal/gamesettings/source"
	_ "github.com/alteredtech/frameshare-collector/internal/gamesettings/standalone"
	_ "github.com/alteredtech/frameshare-collector/internal/gamesettings/unreal"
)
