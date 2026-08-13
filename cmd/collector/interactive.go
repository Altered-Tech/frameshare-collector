package main

import (
	"bufio"
	"fmt"
	"os"

	"golang.org/x/term"
)

type pickerKey int

const (
	keyNone pickerKey = iota
	keyUp
	keyDown
	keyEnter
	keyCancel
)

// interactiveGamePicker renders labels as a menu and lets the user move the
// cursor with the up/down arrow keys, confirming with Enter. Ctrl+C or 'q'
// cancels. It requires stdin to be a real terminal — callers should check
// term.IsTerminal first and fall back to numberedGamePicker otherwise
// (e.g. piped input, or tests, where raw mode isn't available).
func interactiveGamePicker(labels []string, stdin, stdout *os.File) (int, error) {
	fd := int(stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return 0, fmt.Errorf("enter raw terminal mode: %w", err)
	}
	defer term.Restore(fd, oldState)

	// In raw mode, line endings from the app's own writes need an explicit
	// \r before \n, and ISIG is off, so Ctrl+C arrives as byte 0x03 rather
	// than a signal — that's handled as keyCancel below.
	fmt.Fprint(stdout, "Use up/down to choose, Enter to confirm, Ctrl+C to cancel:\r\n")

	cursor := 0
	renderMenu(stdout, labels, cursor)

	reader := bufio.NewReader(stdin)
	for {
		key, err := readKey(reader)
		if err != nil {
			return 0, err
		}
		switch key {
		case keyUp:
			if cursor > 0 {
				cursor--
			}
		case keyDown:
			if cursor < len(labels)-1 {
				cursor++
			}
		case keyEnter:
			return cursor, nil
		case keyCancel:
			return 0, fmt.Errorf("selection canceled")
		default:
			continue
		}
		moveCursorUp(stdout, len(labels))
		renderMenu(stdout, labels, cursor)
	}
}

func renderMenu(w *os.File, labels []string, cursor int) {
	for i, label := range labels {
		prefix := "  "
		if i == cursor {
			prefix = "> "
		}
		// \x1b[2K clears the line before rewriting it, so a shorter label
		// (or one that used to be selected) doesn't leave stray characters.
		fmt.Fprintf(w, "\r\x1b[2K%s%d) %s\r\n", prefix, i+1, label)
	}
}

func moveCursorUp(w *os.File, n int) {
	if n > 0 {
		fmt.Fprintf(w, "\x1b[%dA", n)
	}
}

// readKey reads a single logical keypress from a raw-mode terminal. Arrow
// keys arrive as the 3-byte escape sequence ESC '[' 'A'/'B'; any other
// escape sequence, or a lone ESC with nothing following, is reported as
// keyNone rather than blocking indefinitely on ReadByte.
func readKey(r *bufio.Reader) (pickerKey, error) {
	b, err := r.ReadByte()
	if err != nil {
		return keyNone, err
	}
	switch b {
	case '\r', '\n':
		return keyEnter, nil
	case 0x03, 'q', 'Q':
		return keyCancel, nil
	case 0x1b:
		b2, err := r.ReadByte()
		if err != nil || b2 != '[' {
			return keyNone, nil
		}
		b3, err := r.ReadByte()
		if err != nil {
			return keyNone, nil
		}
		switch b3 {
		case 'A':
			return keyUp, nil
		case 'B':
			return keyDown, nil
		default:
			return keyNone, nil
		}
	default:
		return keyNone, nil
	}
}
