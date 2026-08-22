// Package gamesettings extracts graphics settings (resolution, graphics
// preset, upscaling mode, ray tracing) from a game's local configuration
// files. It follows the Snapshot/DeviceInfo pattern used in
// internal/hardware/snapshot.go: GameProfile is the full per-title report,
// TitleSettings is the settings payload within it.
//
// Each supported title implements Parser and registers it (via Register)
// from a subpackage grouped by engine/config format -- unreal, source,
// standalone -- rather than piling every title's file into this package
// directly, since the supported-title list is expected to grow into the
// hundreds; see internal/gamesettings/all to pull in every one of them at
// once. Collect finds the right Parser for an installed game and runs it.
// A title with no registered Parser yields ErrTitleUnsupported -- the
// initial set covers a handful of titles chosen for having accessible,
// well-documented config formats, not every installed game.
package gamesettings

import "time"

// GameProfile is the graphics settings extracted for a single installed
// title, keyed to the library entry (see internal/library.Game) it was
// parsed from.
type GameProfile struct {
	ParserVersion string    `json:"parser_version"`
	ParsedAt      time.Time `json:"parsed_at"`
	AppID         string    `json:"app_id,omitempty"`
	Name          string    `json:"name"`
	Source        string    `json:"source"` // e.g. "steam"; matches library.Source
	// ProtonVersion is the Proton compatibility tool Steam ran this title
	// with, populated only when ConfigPath was actually resolved from
	// inside a Proton prefix (see ProtonVersion in path.go) -- empty for
	// native and non-Linux profiles.
	ProtonVersion string `json:"proton_version,omitempty"`
	// ConfigPath is the local file the settings below were read from. It's
	// excluded from JSON deliberately: under the user's home directory (the
	// common case -- e.g. ~/Library/Preferences/... on macOS), it embeds
	// their OS username, and a GameProfile is meant to be shared, not kept
	// local like the raw hardware snapshot. It's still populated for
	// in-process use (logging, error messages).
	ConfigPath string        `json:"-"`
	Settings   TitleSettings `json:"settings"`
}

// TitleSettings is the full set of graphics settings collected for a
// title. Every field is best-effort: per-title parsers (see issue #3)
// populate whatever the title's config format actually exposes, and a
// title that doesn't expose a given knob (e.g. no ray tracing support)
// simply leaves that field at its zero value. GraphicsPreset captures the
// title's own overall quality preset, if it has one, independent of the
// individual Detail fields, since some titles only expose a preset and
// others only expose individual sliders.
type TitleSettings struct {
	Display        DisplaySettings    `json:"display"`
	GraphicsPreset string             `json:"graphics_preset,omitempty"` // e.g. "Low", "High", "Custom"
	Detail         DetailSettings     `json:"detail"`
	Upscaling      UpscalingSettings  `json:"upscaling"`
	RayTracing     RayTracingSettings `json:"ray_tracing"`
}

// Resolution is a render or output resolution.
type Resolution struct {
	WidthPx  int `json:"width_px"`
	HeightPx int `json:"height_px"`
}

// DisplaySettings covers output/presentation settings, as opposed to the
// in-scene rendering detail in DetailSettings.
type DisplaySettings struct {
	Resolution Resolution `json:"resolution"`
	// RefreshRateHz is the display refresh rate the title is targeting,
	// distinct from the achieved frame rate reported alongside a profile.
	RefreshRateHz  float64    `json:"refresh_rate_hz,omitempty"`
	WindowMode     WindowMode `json:"window_mode,omitempty"`
	VSync          bool       `json:"vsync"`
	FrameRateLimit int        `json:"frame_rate_limit,omitempty"` // FPS cap; 0 = uncapped/not set
	HDR            bool       `json:"hdr"`
}

// WindowMode identifies how a title presents its window.
type WindowMode string

const (
	WindowFullscreen WindowMode = "fullscreen"
	WindowBorderless WindowMode = "borderless"
	WindowWindowed   WindowMode = "windowed"
)

// DetailSettings covers the in-scene rendering quality knobs many titles
// expose individually, in addition to (or instead of) a single overall
// GraphicsPreset.
type DetailSettings struct {
	TextureQuality       string `json:"texture_quality,omitempty"`
	ShadowQuality        string `json:"shadow_quality,omitempty"`
	AntiAliasing         string `json:"anti_aliasing,omitempty"` // e.g. "TAA", "FXAA", "SMAA", "DLAA", "Off"
	AmbientOcclusion     string `json:"ambient_occlusion,omitempty"`
	ReflectionQuality    string `json:"reflection_quality,omitempty"`
	ViewDistance         string `json:"view_distance,omitempty"` // aka level-of-detail distance
	FoliageDensity       string `json:"foliage_density,omitempty"`
	EffectsQuality       string `json:"effects_quality,omitempty"`       // particles, volumetrics, etc.
	AnisotropicFiltering int    `json:"anisotropic_filtering,omitempty"` // 1/2/4/8/16
	MotionBlur           bool   `json:"motion_blur"`
	DepthOfField         bool   `json:"depth_of_field"`
	Sharpening           int    `json:"sharpening,omitempty"` // 0-100
}

// UpscalingSettings covers AI/spatial upscaling and frame generation.
type UpscalingSettings struct {
	Mode UpscalingMode `json:"mode,omitempty"`
	// QualityLevel is the upscaler's own tier, e.g. "Quality", "Balanced",
	// "Performance", "Ultra Performance" — named differently per vendor
	// but consistent enough in practice to store as free text.
	QualityLevel      string `json:"quality_level,omitempty"`
	FrameGeneration   bool   `json:"frame_generation"`
	DynamicResolution bool   `json:"dynamic_resolution"`
}

// UpscalingMode identifies which upscaling technology, if any, a title is
// configured to use.
type UpscalingMode string

const (
	UpscalingNone    UpscalingMode = "none"
	UpscalingDLSS    UpscalingMode = "dlss"
	UpscalingFSR     UpscalingMode = "fsr"
	UpscalingXeSS    UpscalingMode = "xess"
	UpscalingTSR     UpscalingMode = "tsr" // Unreal Engine's built-in Temporal Super Resolution
	UpscalingUnknown UpscalingMode = "unknown"
)

// RayTracingSettings covers ray-traced effects. Enabled is the overall
// toggle called out in issue #1; the per-effect fields are populated only
// for titles that expose ray-traced reflections/shadows/GI/AO as separate
// switches rather than one combined toggle.
type RayTracingSettings struct {
	Enabled bool `json:"enabled"`
	// Preset is the title's own ray tracing quality tier, if it has one
	// (e.g. "Low", "Ultra", "Psycho"), independent of the per-effect flags.
	Preset             string `json:"preset,omitempty"`
	Reflections        bool   `json:"reflections,omitempty"`
	Shadows            bool   `json:"shadows,omitempty"`
	GlobalIllumination bool   `json:"global_illumination,omitempty"`
	AmbientOcclusion   bool   `json:"ambient_occlusion,omitempty"`
}
