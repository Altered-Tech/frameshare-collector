package profile

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"time"
)

// Field is one leaf value in a Profile, flattened for display and (later)
// edit in a controller-driven review screen. Path is a stable identifier
// built from the underlying Go field names (e.g.
// "hardware.CPU.PhysicalCores") and doubles as the key into
// Profile.Overrides; Label is the human-readable form of that same path
// for display (e.g. "CPU Physical Cores").
type Field struct {
	Path       string
	Label      string
	Value      string
	Overridden bool
}

// Fields flattens p into an ordered list of leaf Fields for display: every
// exported field of Hardware, then (if set) every exported field of
// GameSettings.
//
// It walks the struct tree via reflection rather than hand-listing every
// hardware/game-settings field, so newly added detection fields --
// internal/gamesettings's own package doc expects its supported-title
// list to grow into the hundreds, each adding settings fields over time
// -- automatically show up on the review screen without this package
// needing a matching update. A hand-maintained field list would instead
// silently omit new fields from review, which is the same class of gap
// issue #23 already flags for detection failures: a user approving a
// profile that looks complete but isn't.
func (p *Profile) Fields() []Field {
	var out []Field
	walk("hardware", reflect.ValueOf(p.Hardware), &out, p.Overrides)
	if p.GameSettings != nil {
		walk("game_settings", reflect.ValueOf(*p.GameSettings), &out, p.Overrides)
	}
	return out
}

var timeType = reflect.TypeOf(time.Time{})

// walk appends one Field per leaf value reachable from v, using path as
// the accumulated dotted/indexed breadcrumb of raw Go field names (also
// used as each Field's Path); Label is derived from path separately, at
// the leaf, so Path stays a stable machine key even as Label's
// humanization rules evolve.
func walk(path string, v reflect.Value, out *[]Field, overrides map[string]bool) {
	if v.Type() == timeType {
		appendLeaf(path, v, out, overrides)
		return
	}

	switch v.Kind() {
	case reflect.Struct:
		t := v.Type()
		for i := 0; i < t.NumField(); i++ {
			sf := t.Field(i)
			if !sf.IsExported() {
				continue
			}
			// A field tagged json:"-" is deliberately excluded from the
			// serialized snapshot (e.g. GameProfile.ConfigPath, which
			// embeds the user's OS username) -- exclude it from review
			// for the same reason.
			if sf.Tag.Get("json") == "-" {
				continue
			}
			walk(path+"."+sf.Name, v.Field(i), out, overrides)
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			walk(fmt.Sprintf("%s[%d]", path, i+1), v.Index(i), out, overrides)
		}
	case reflect.Ptr, reflect.Interface:
		if v.IsNil() {
			return
		}
		walk(path, v.Elem(), out, overrides)
	default:
		appendLeaf(path, v, out, overrides)
	}
}

func appendLeaf(path string, v reflect.Value, out *[]Field, overrides map[string]bool) {
	*out = append(*out, Field{
		Path:       path,
		Label:      labelFromPath(path),
		Value:      formatValue(v),
		Overridden: overrides[path],
	})
}

// labelFromPath turns a raw Path like "hardware.CPU.PhysicalCores" into a
// display label "CPU Physical Cores": it drops the leading section name
// (the caller's List already groups rows by section) and humanizes each
// remaining segment.
func labelFromPath(path string) string {
	parts := strings.Split(path, ".")
	if len(parts) > 1 {
		parts = parts[1:]
	}
	for i, part := range parts {
		parts[i] = humanize(part)
	}
	return strings.Join(parts, " ")
}

func formatValue(v reflect.Value) string {
	if v.Type() == timeType {
		t := v.Interface().(time.Time)
		if t.IsZero() {
			return ""
		}
		return t.Format("2006-01-02 15:04:05 MST")
	}
	switch v.Kind() {
	case reflect.Bool:
		if v.Bool() {
			return "Yes"
		}
		return "No"
	case reflect.String:
		return v.String()
	default:
		return fmt.Sprintf("%v", v.Interface())
	}
}

// lowerUpper matches a lowercase-then-uppercase boundary (e.g. the "xM"
// in "MaxMHz"); acronymBoundary matches the end of an uppercase run
// followed by a new capitalized word of at least two more lowercase
// letters (e.g. the "SD" in "IsOSDrive" splitting to "OS Drive"). The
// two-letter minimum on the trailing word keeps short acronym suffixes
// like the "s" in "GPUs" or the "Hz" in "MHz" from being misread as the
// start of a new word and split off on their own.
var (
	lowerUpper      = regexp.MustCompile(`([a-z0-9])([A-Z])`)
	acronymBoundary = regexp.MustCompile(`([A-Z]+)([A-Z][a-z]{2,})`)
)

func humanize(fieldName string) string {
	s := lowerUpper.ReplaceAllString(fieldName, "$1 $2")
	s = acronymBoundary.ReplaceAllString(s, "$1 $2")
	return s
}
