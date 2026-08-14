package gamesettings

import (
	"bufio"
	"bytes"
	"strings"
)

// parseINI parses a minimal INI file: `[Section]` headers followed by
// `key=value` lines, as used by Unreal Engine's GameUserSettings.ini and
// similar per-title config files. Comment lines starting with `;` or `#`
// and blank lines are skipped. Keys that appear before the first section
// header are stored under the "" section. Duplicate keys within a section
// keep the last value seen, matching how Unreal itself reads its own INI
// files.
func parseINI(data []byte) map[string]map[string]string {
	sections := map[string]map[string]string{"": {}}
	section := ""

	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.TrimSuffix(strings.TrimPrefix(line, "["), "]")
			if _, ok := sections[section]; !ok {
				sections[section] = map[string]string{}
			}
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		sections[section][strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	return sections
}

// iniSectionWithSuffix returns the first section whose name ends with
// suffix, along with whether one was found. Unreal Engine names a title's
// main settings section after its studio-specific GameUserSettings
// subclass (e.g. "/Script/Hk_project.HKGameUserSettings"), which varies
// per title, but it always ends in "GameUserSettings" -- matching on that
// suffix avoids hardcoding the title-specific prefix.
func iniSectionWithSuffix(sections map[string]map[string]string, suffix string) (map[string]string, bool) {
	for name, kv := range sections {
		if strings.HasSuffix(name, suffix) {
			return kv, true
		}
	}
	return nil, false
}
