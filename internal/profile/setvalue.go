package profile

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// SetValue parses value and writes it into the leaf field at path (a
// Path from Fields, e.g. "hardware.CPU.PhysicalCores" or
// "hardware.GPUs[1].Name"), then marks that path Overridden. It returns
// an error, leaving the Profile unchanged, if path doesn't resolve to a
// settable leaf or value can't be parsed as that leaf's type (e.g. "not
// a number" for an int field) -- an edit that can't be applied should be
// rejected visibly, not silently discarded or used to corrupt the field
// with a mismatched type.
//
// Time fields (CollectedAt, ParsedAt) aren't editable through this: they
// record when detection ran, not a detected setting, so resolvePath
// rejects them.
func (p *Profile) SetValue(path, value string) error {
	target, err := p.resolvePath(path)
	if err != nil {
		return err
	}
	before := formatValue(target)
	if err := setLeaf(target, value); err != nil {
		return fmt.Errorf("set %s: %w", path, err)
	}
	// Only mark path overridden if the value actually changed: otherwise
	// re-submitting a field's pre-filled text unedited (the common case
	// when a user opens the edit widget just to look, then hits Save)
	// would spuriously flag it as user-corrected.
	if formatValue(target) != before {
		if p.Overrides == nil {
			p.Overrides = map[string]bool{}
		}
		p.Overrides[path] = true
	}
	return nil
}

// Editable reports whether path (a Path from Fields) can be changed via
// SetValue. It's false for timestamps and for any path that doesn't
// resolve at all; every other leaf Fields can produce is a supported
// scalar kind (walk only ever calls appendLeaf, and therefore only ever
// produces a Path, for kinds SetValue's setLeaf handles).
func (p *Profile) Editable(path string) bool {
	v, err := p.resolvePath(path)
	return err == nil && v.IsValid()
}

// resolvePath walks p's Hardware (or GameSettings, if the path starts
// with "game_settings.") struct following the same field-name/index
// scheme walk uses to build Path in the first place, returning an
// addressable, settable reflect.Value for the leaf.
func (p *Profile) resolvePath(path string) (reflect.Value, error) {
	var root reflect.Value
	var rest string
	switch {
	case strings.HasPrefix(path, "hardware."):
		root = reflect.ValueOf(&p.Hardware).Elem()
		rest = strings.TrimPrefix(path, "hardware.")
	case strings.HasPrefix(path, "game_settings."):
		if p.GameSettings == nil {
			return reflect.Value{}, fmt.Errorf("no game settings on this profile")
		}
		root = reflect.ValueOf(p.GameSettings).Elem()
		rest = strings.TrimPrefix(path, "game_settings.")
	default:
		return reflect.Value{}, fmt.Errorf("unrecognized path %q", path)
	}

	v := root
	for _, segment := range strings.Split(rest, ".") {
		name, index, hasIndex := parseSegment(segment)

		v = v.FieldByName(name)
		if !v.IsValid() {
			return reflect.Value{}, fmt.Errorf("no field %q in path %q", name, path)
		}
		if hasIndex {
			if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
				return reflect.Value{}, fmt.Errorf("%q is not indexable in path %q", name, path)
			}
			if index < 1 || index > v.Len() {
				return reflect.Value{}, fmt.Errorf("index %d out of range for %q in path %q", index, name, path)
			}
			v = v.Index(index - 1)
		}
	}

	if v.Type() == timeType {
		return reflect.Value{}, fmt.Errorf("%q is a timestamp, not editable", path)
	}
	if !v.CanSet() {
		return reflect.Value{}, fmt.Errorf("%q is not settable", path)
	}
	return v, nil
}

// parseSegment splits a path segment like "GPUs[1]" into its field name
// "GPUs" and 1-based index 1, matching how walk builds indexed segments.
func parseSegment(segment string) (name string, index int, hasIndex bool) {
	open := strings.IndexByte(segment, '[')
	if open < 0 {
		return segment, 0, false
	}
	name = segment[:open]
	idxStr := strings.TrimSuffix(segment[open+1:], "]")
	n, err := strconv.Atoi(idxStr)
	if err != nil {
		return segment, 0, false
	}
	return name, n, true
}

// parseBool accepts the yes/no formatValue renders bools as, in addition
// to a few common spellings, so re-submitting a bool field's pre-filled
// display text unchanged round-trips instead of failing to parse.
func parseBool(s string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "yes", "y", "true", "on", "1":
		return true, nil
	case "no", "n", "false", "off", "0":
		return false, nil
	default:
		return false, fmt.Errorf("not a recognized yes/no value")
	}
}

// setLeaf parses value into v's underlying type and sets it. v must be
// addressable and settable (see resolvePath).
func setLeaf(v reflect.Value, value string) error {
	switch v.Kind() {
	case reflect.String:
		v.SetString(value)
	case reflect.Bool:
		b, err := parseBool(value)
		if err != nil {
			return fmt.Errorf("%q is not yes/no", value)
		}
		v.SetBool(b)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fmt.Errorf("%q is not a whole number", value)
		}
		v.SetInt(n)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return fmt.Errorf("%q is not a non-negative whole number", value)
		}
		v.SetUint(n)
	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("%q is not a number", value)
		}
		v.SetFloat(f)
	default:
		return fmt.Errorf("unsupported field type %s", v.Kind())
	}
	return nil
}
