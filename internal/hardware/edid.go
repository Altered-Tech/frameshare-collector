package hardware

import (
	"bytes"
	"strings"
)

// edidHeader is the fixed 8-byte magic every EDID block starts with.
var edidHeader = []byte{0x00, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0x00}

const (
	edidMinLength        = 128
	edidDescriptorStart  = 54
	edidDescriptorSize   = 18
	edidDescriptorCount  = 4
	edidMonitorNameTag   = 0xFC
	edidTextDescPrefix   = 5 // descriptor text starts at byte offset 5 within the 18-byte block
	edidTextDescTermByte = 0x0A
)

// decodeEDID extracts the manufacturer PNP ID and monitor model name from a
// raw EDID block, per the VESA E-EDID spec: bytes 8-9 are the packed
// manufacturer ID (section 3.4), and display descriptor blocks at offsets
// 54/72/90/108 carry the ASCII monitor name under tag 0xFC (section
// 3.10.3.4). Returns empty strings if data isn't a valid/recognized EDID.
func decodeEDID(data []byte) (pnpID, model string) {
	if len(data) < edidMinLength || !bytes.Equal(data[:8], edidHeader) {
		return "", ""
	}

	pnpID = decodeEDIDPNPID(data[8], data[9])

	for i := 0; i < edidDescriptorCount; i++ {
		off := edidDescriptorStart + i*edidDescriptorSize
		block := data[off : off+edidDescriptorSize]
		// Detailed timing descriptors have a non-zero pixel clock in
		// bytes 0-1; display descriptors (which include the monitor
		// name) always have 0x0000 there.
		if block[0] != 0 || block[1] != 0 {
			continue
		}
		if block[3] == edidMonitorNameTag {
			model = decodeEDIDText(block[edidTextDescPrefix:edidDescriptorSize])
			break
		}
	}
	return pnpID, model
}

// decodeEDIDPNPID unpacks the 5-bit-per-letter manufacturer ID from EDID
// bytes 8-9 (big-endian) into its 3-letter PNP ID, e.g. "DEL".
func decodeEDIDPNPID(b8, b9 byte) string {
	v := uint16(b8)<<8 | uint16(b9)
	letters := [3]byte{
		byte((v>>10)&0x1F) + 'A' - 1,
		byte((v>>5)&0x1F) + 'A' - 1,
		byte(v&0x1F) + 'A' - 1,
	}
	for _, c := range letters {
		if c < 'A' || c > 'Z' {
			return ""
		}
	}
	return string(letters[:])
}

// decodeEDIDText trims the 0x0A terminator (and anything after it, e.g.
// trailing 0x20 padding) from an EDID descriptor text field.
func decodeEDIDText(b []byte) string {
	if i := bytes.IndexByte(b, edidTextDescTermByte); i >= 0 {
		b = b[:i]
	}
	return strings.TrimRight(string(b), " ")
}
