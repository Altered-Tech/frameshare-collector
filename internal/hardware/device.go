package hardware

import "strings"

// handheldSignature matches a known gaming handheld/dedicated device by
// looking for a set of keywords in the combined lowercased DMI
// vendor+model string. Every keyword must be present.
type handheldSignature struct {
	name     string
	keywords []string
}

// knownHandhelds is a best-effort, manually maintained list of gaming
// handhelds/consoles identifiable via DMI/SMBIOS strings. New devices
// (Legion Go successors, the upcoming Steam Machine, etc) should be added
// here once their DMI vendor/product strings are known.
var knownHandhelds = []handheldSignature{
	{name: "Steam Deck LCD", keywords: []string{"valve", "jupiter"}},
	{name: "Steam Deck OLED", keywords: []string{"valve", "galileo"}},
	{name: "Steam Machine", keywords: []string{"valve", "fremont"}},
	{name: "ROG Ally X", keywords: []string{"rog ally x"}},
	{name: "ROG Ally", keywords: []string{"rog ally"}},
	{name: "Lenovo Legion Go", keywords: []string{"legion go"}},
	{name: "MSI Claw", keywords: []string{"msi", "claw"}},
	{name: "AYANEO", keywords: []string{"ayaneo"}},
	{name: "GPD Win", keywords: []string{"gpd", "win"}},
	{name: "OneXPlayer", keywords: []string{"onexplayer"}},
}

// matchHandheld returns the display name of a known handheld device given
// its DMI vendor and model/product strings, or "" if unrecognized.
func matchHandheld(vendor, model string) string {
	haystack := strings.ToLower(vendor + " " + model)
	for _, sig := range knownHandhelds {
		matched := true
		for _, kw := range sig.keywords {
			if !strings.Contains(haystack, kw) {
				matched = false
				break
			}
		}
		if matched {
			return sig.name
		}
	}
	return ""
}
