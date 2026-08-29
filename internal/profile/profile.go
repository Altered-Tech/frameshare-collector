// Package profile merges a hardware.Snapshot and an optional
// gamesettings.GameProfile into the single Profile a user reviews, edits,
// and confirms before it's saved (see cmd/gui) -- unlike its two source
// types, which are raw, unreviewed detection output written straight to
// disk by cmd/collector.
package profile

import (
	"fmt"
	"time"

	"github.com/alteredtech/frameshare-collector/internal/gamesettings"
	"github.com/alteredtech/frameshare-collector/internal/hardware"
	"github.com/alteredtech/frameshare-collector/internal/library"
)

// Profile is the merged hardware + game-settings report a user reviews
// before it's saved. GameSettings is nil when no game was selected, or
// the selected title has no registered settings parser.
type Profile struct {
	CollectorVersion string                    `json:"collector_version"`
	CollectedAt      time.Time                 `json:"collected_at"`
	Hardware         hardware.Snapshot         `json:"hardware"`
	GameSettings     *gamesettings.GameProfile `json:"game_settings,omitempty"`
	// Overrides records the Path (see Field) of every value the user has
	// edited away from what was detected, so a saved Profile still shows
	// provenance rather than looking identically "detected" everywhere.
	Overrides map[string]bool `json:"overrides,omitempty"`
}

// Merge combines a hardware snapshot with an optional game settings
// profile (nil when the user reviewed hardware only, with no game
// selected, or the selected title has no registered parser) into a
// single reviewable Profile.
func Merge(hw hardware.Snapshot, gs *gamesettings.GameProfile) *Profile {
	return &Profile{
		CollectorVersion: hw.CollectorVersion,
		CollectedAt:      hw.CollectedAt,
		Hardware:         hw,
		GameSettings:     gs,
		Overrides:        map[string]bool{},
	}
}

// CollectGameSettings runs the settings parser registered for game, if
// any. Mirroring cmd/collector's tolerance for missing OS tooling, an
// unsupported title or an unreadable config file (most commonly because
// the game has never been launched, so it hasn't written one yet) isn't
// fatal -- it's reported via warn and Merge is meant to be called with a
// nil GameProfile in that case. warn receives one line describing why
// settings weren't collected; pass nil to discard it.
func CollectGameSettings(game library.Game, source library.Source, warn func(string)) *gamesettings.GameProfile {
	if warn == nil {
		warn = func(string) {}
	}
	if !gamesettings.Supported(game, source) {
		warn(fmt.Sprintf("no settings parser registered for %s, skipping game settings", game.Name))
		return nil
	}
	p, err := gamesettings.Collect(game, source)
	if err != nil {
		warn(fmt.Sprintf("could not collect settings for %s: %v", game.Name, err))
		return nil
	}
	return p
}
