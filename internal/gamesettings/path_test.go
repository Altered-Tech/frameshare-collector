package gamesettings

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/alteredtech/frameshare-collector/internal/library"
)

func TestProtonAppDataPath(t *testing.T) {
	steamLibrary := t.TempDir()
	installPath := filepath.Join(steamLibrary, "steamapps", "common", "Stray")
	game := library.Game{Name: "Stray", InstallPath: installPath, AppID: "1332010"}

	got, err := ProtonAppDataPath(game, filepath.Join("Stray", "Saved", "Config", "WindowsNoEditor", "GameUserSettings.ini"))
	if err != nil {
		t.Fatalf("ProtonAppDataPath() error = %v", err)
	}
	want := filepath.Join(steamLibrary, "steamapps", "compatdata", "1332010", "pfx", "drive_c", "users", "steamuser", "AppData", "Local", "Stray", "Saved", "Config", "WindowsNoEditor", "GameUserSettings.ini")
	if got != want {
		t.Errorf("ProtonAppDataPath() = %q, want %q", got, want)
	}
}

func TestProtonAppDataPathNoAppID(t *testing.T) {
	game := library.Game{Name: "Stray", InstallPath: filepath.Join(t.TempDir(), "steamapps", "common", "Stray")}
	if _, err := ProtonAppDataPath(game, "irrelevant"); err == nil {
		t.Error("ProtonAppDataPath() error = nil, want error for missing app id")
	}
}

// configVDFFixture is a trimmed synthetic config.vdf with a single
// CompatToolMapping entry, matching the shape Steam itself writes under
// InstallConfigStore/Software/Valve/Steam/CompatToolMapping/<appid>.
const configVDFFixture = `"InstallConfigStore"
{
	"Software"
	{
		"Valve"
		{
			"Steam"
			{
				"CompatToolMapping"
				{
					"1332010"
					{
						"name"		"proton_experimental"
						"config"		""
						"priority"		"250"
					}
				}
			}
		}
	}
}
`

// setUpProtonPrefix creates a synthetic Steam root under t.TempDir() with,
// if withConfig, a config.vdf mapping appID to the Proton build in
// configVDFFixture. It returns the Game whose InstallPath points into
// that root, plus the config path ProtonAppDataPath would resolve for it
// -- a stand-in for the path a Parser would actually read the title's
// settings from.
func setUpProtonPrefix(t *testing.T, appID string, withConfig bool) (library.Game, string) {
	t.Helper()
	steamRoot := t.TempDir()
	installPath := filepath.Join(steamRoot, "steamapps", "common", "Stray")
	game := library.Game{Name: "Stray", InstallPath: installPath, AppID: appID}

	if withConfig {
		configDir := filepath.Join(steamRoot, "config")
		if err := os.MkdirAll(configDir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(configDir, "config.vdf"), []byte(configVDFFixture), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	configPath, err := ProtonAppDataPath(game, filepath.Join("Stray", "Saved", "Config", "WindowsNoEditor", "GameUserSettings.ini"))
	if err != nil {
		t.Fatal(err)
	}
	return game, configPath
}

func TestProtonVersion(t *testing.T) {
	game, configPath := setUpProtonPrefix(t, "1332010", true)
	if got, want := ProtonVersion(game, configPath), "proton_experimental"; got != want {
		t.Errorf("ProtonVersion() = %q, want %q", got, want)
	}
}

// TestProtonVersionConfigPathNotUnderPrefix covers a title with a native
// Linux build (e.g. Terraria) that also has a stale compatdata prefix
// and CompatToolMapping entry -- from a compatibility tool override the
// user forced at some point in Steam, even though it wasn't used to
// resolve this profile's settings. ProtonVersion must not report a
// version in that case: the config path a Parser actually resolved
// (configPath here) is what determines whether Proton was involved, not
// merely whether Steam has *a* mapping on file for the app id.
func TestProtonVersionConfigPathNotUnderPrefix(t *testing.T) {
	game, _ := setUpProtonPrefix(t, "1332010", true)
	nativeConfigPath := filepath.Join(t.TempDir(), ".local", "share", "Terraria", "config.json")
	if got := ProtonVersion(game, nativeConfigPath); got != "" {
		t.Errorf("ProtonVersion() = %q, want empty when configPath isn't under the Proton prefix", got)
	}
}

func TestProtonVersionMissingConfigVDF(t *testing.T) {
	// A fresh Steam install may have a Proton prefix but no config.vdf yet.
	game, configPath := setUpProtonPrefix(t, "1332010", false)
	if got := ProtonVersion(game, configPath); got != "" {
		t.Errorf("ProtonVersion() = %q, want empty when config.vdf is missing", got)
	}
}

func TestProtonVersionNoMappingForAppID(t *testing.T) {
	// configPath is under the Proton prefix and config.vdf exists, but
	// it has no mapping for this app id -- e.g. Steam never recorded a
	// compat tool choice.
	game, configPath := setUpProtonPrefix(t, "999999", true)
	if got := ProtonVersion(game, configPath); got != "" {
		t.Errorf("ProtonVersion() = %q, want empty when config.vdf has no mapping for this app id", got)
	}
}

func TestProtonVersionNoAppID(t *testing.T) {
	game := library.Game{Name: "Stray", InstallPath: filepath.Join(t.TempDir(), "steamapps", "common", "Stray")}
	if got := ProtonVersion(game, "irrelevant"); got != "" {
		t.Errorf("ProtonVersion() = %q, want empty for a game with no app id", got)
	}
}
