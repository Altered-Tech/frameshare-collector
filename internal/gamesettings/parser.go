package gamesettings

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/alteredtech/frameshare-collector/internal/library"
)

// ParserVersion identifies the field-mapping logic that produced a
// GameProfile. Bump it whenever a parser's mapping from a title's config
// keys to TitleSettings changes, so consumers can tell profiles collected
// under an old mapping apart from new ones.
const ParserVersion = "1"

// ErrTitleUnsupported is returned by Collect when no registered Parser
// matches the given game.
var ErrTitleUnsupported = errors.New("gamesettings: no parser for this title")

// Parser extracts TitleSettings from one title's local config file. Titles
// are grouped by engine/format into their own subpackages (unreal,
// source, standalone -- see internal/gamesettings/all for the full list);
// each title implements Parser and registers itself via Register() in its
// package's init().
type Parser interface {
	// Matches reports whether this parser handles game.
	Matches(game library.Game, source library.Source) bool
	// ConfigPath returns the absolute path to the config file this
	// parser reads for game. The file isn't guaranteed to exist yet --
	// e.g. the title may never have been launched, so it hasn't
	// written one.
	ConfigPath(game library.Game) (string, error)
	// Parse decodes a config file's contents into TitleSettings.
	Parse(data []byte) (TitleSettings, error)
}

// parsers is the set of registered per-title Parsers, populated by each
// title package's init() calling Register.
var parsers []Parser

// Register adds p to the set of Parsers Collect and Supported search.
// Title packages call this from an init() function; see
// internal/gamesettings/unreal/stray.go for an example.
func Register(p Parser) {
	parsers = append(parsers, p)
}

// Supported reports whether any registered Parser handles game, without
// reading its config file.
func Supported(game library.Game, source library.Source) bool {
	return find(game, source) != nil
}

// Collect finds the registered Parser for game, reads its config file, and
// returns the resulting GameProfile. It returns ErrTitleUnsupported if no
// parser matches game; if a parser matches but its config file can't be
// read or understood -- most commonly because the title has never been
// launched, so it hasn't written one yet -- it returns a wrapped error
// instead.
func Collect(game library.Game, source library.Source) (*GameProfile, error) {
	p := find(game, source)
	if p == nil {
		return nil, fmt.Errorf("%w: %s", ErrTitleUnsupported, game.Name)
	}

	path, err := p.ConfigPath(game)
	if err != nil {
		return nil, fmt.Errorf("gamesettings: %s: %w", game.Name, err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("gamesettings: %s: reading config at %s: %w", game.Name, path, err)
	}
	settings, err := p.Parse(data)
	if err != nil {
		return nil, fmt.Errorf("gamesettings: %s: parsing config at %s: %w", game.Name, path, err)
	}

	return &GameProfile{
		ParserVersion: ParserVersion,
		ParsedAt:      time.Now().UTC(),
		AppID:         game.AppID,
		Name:          game.Name,
		Source:        string(source),
		ConfigPath:    path,
		Settings:      settings,
	}, nil
}

func find(game library.Game, source library.Source) Parser {
	for _, p := range parsers {
		if p.Matches(game, source) {
			return p
		}
	}
	return nil
}
