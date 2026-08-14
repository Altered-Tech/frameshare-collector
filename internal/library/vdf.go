package library

import (
	"fmt"
	"strings"
)

// ParseVDF parses Valve's KeyValues ("VDF") text format, used for
// steamapps/libraryfolders.vdf and steamapps/appmanifest_*.acf here, and
// more generally by other Source-engine config files (e.g. a title's
// cfg/video.txt) that internal/gamesettings parsers read. The format is a
// tree of quoted-string keys mapped to either another quoted string or a
// brace-delimited nested object:
//
//	"AppState"
//	{
//		"appid"		"228980"
//		"name"		"Steamworks Common Redistributables"
//	}
//
// Values in the returned map are either string or map[string]any.
func ParseVDF(data []byte) (map[string]any, error) {
	p := &vdfParser{data: data}
	obj, err := p.readObjectBody(false)
	if err != nil {
		return nil, err
	}
	return obj, nil
}

type vdfParser struct {
	data []byte
	pos  int
}

// readObjectBody reads key/value pairs until it hits a closing '}' (when
// insideBraces is true) or the end of input (top level).
func (p *vdfParser) readObjectBody(insideBraces bool) (map[string]any, error) {
	obj := map[string]any{}
	for {
		p.skipWhitespace()
		if p.pos >= len(p.data) {
			if insideBraces {
				return nil, fmt.Errorf("vdf: unexpected end of input, unclosed object")
			}
			return obj, nil
		}
		if p.data[p.pos] == '}' {
			if !insideBraces {
				return nil, fmt.Errorf("vdf: unexpected '}' at offset %d", p.pos)
			}
			p.pos++
			return obj, nil
		}

		key, err := p.readString()
		if err != nil {
			return nil, err
		}
		p.skipWhitespace()
		if p.pos < len(p.data) && p.data[p.pos] == '{' {
			p.pos++
			child, err := p.readObjectBody(true)
			if err != nil {
				return nil, err
			}
			obj[key] = child
			continue
		}
		val, err := p.readString()
		if err != nil {
			return nil, err
		}
		obj[key] = val
	}
}

// readString reads a double-quoted string, unescaping \" and \\ (the only
// escapes Steam's VDF writer uses).
func (p *vdfParser) readString() (string, error) {
	p.skipWhitespace()
	if p.pos >= len(p.data) || p.data[p.pos] != '"' {
		return "", fmt.Errorf("vdf: expected quoted string at offset %d", p.pos)
	}
	p.pos++

	var sb strings.Builder
	for p.pos < len(p.data) {
		c := p.data[p.pos]
		switch {
		case c == '\\' && p.pos+1 < len(p.data):
			sb.WriteByte(p.data[p.pos+1])
			p.pos += 2
		case c == '"':
			p.pos++
			return sb.String(), nil
		default:
			sb.WriteByte(c)
			p.pos++
		}
	}
	return "", fmt.Errorf("vdf: unterminated string starting near offset %d", p.pos)
}

func (p *vdfParser) skipWhitespace() {
	for p.pos < len(p.data) {
		switch p.data[p.pos] {
		case ' ', '\t', '\r', '\n':
			p.pos++
		default:
			return
		}
	}
}
