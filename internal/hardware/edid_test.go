package hardware

import "testing"

// buildTestEDID assembles a minimal valid 128-byte EDID block with the
// given manufacturer ID bytes (EDID offsets 8-9) and an optional monitor
// name descriptor (tag 0xFC) at offset 54.
func buildTestEDID(mfgHi, mfgLo byte, monitorName string) []byte {
	data := make([]byte, edidMinLength)
	copy(data, edidHeader)
	data[8] = mfgHi
	data[9] = mfgLo

	if monitorName != "" {
		block := data[edidDescriptorStart : edidDescriptorStart+edidDescriptorSize]
		block[3] = edidMonitorNameTag
		text := block[edidTextDescPrefix:edidDescriptorSize]
		n := copy(text, monitorName)
		if n < len(text) {
			text[n] = edidTextDescTermByte
			for i := n + 1; i < len(text); i++ {
				text[i] = ' '
			}
		}
	}
	return data
}

func TestDecodeEDID(t *testing.T) {
	// 0x10AC is Dell's real-world PNP ID ("DEL").
	data := buildTestEDID(0x10, 0xAC, "AW3225QF")

	pnpID, model := decodeEDID(data)
	if pnpID != "DEL" {
		t.Errorf("pnpID = %q, want %q", pnpID, "DEL")
	}
	if model != "AW3225QF" {
		t.Errorf("model = %q, want %q", model, "AW3225QF")
	}
}

func TestDecodeEDIDNoNameDescriptor(t *testing.T) {
	data := buildTestEDID(0x10, 0xAC, "")

	pnpID, model := decodeEDID(data)
	if pnpID != "DEL" {
		t.Errorf("pnpID = %q, want %q", pnpID, "DEL")
	}
	if model != "" {
		t.Errorf("model = %q, want empty", model)
	}
}

func TestDecodeEDIDInvalid(t *testing.T) {
	cases := map[string][]byte{
		"too short":  {0x00, 0xFF, 0xFF},
		"bad header": append([]byte{0x01, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0x00}, make([]byte, edidMinLength-8)...),
		"nil":        nil,
	}
	for name, data := range cases {
		t.Run(name, func(t *testing.T) {
			pnpID, model := decodeEDID(data)
			if pnpID != "" || model != "" {
				t.Errorf("decodeEDID(%s) = (%q, %q), want empty", name, pnpID, model)
			}
		})
	}
}

func TestDecodeEDIDPNPID(t *testing.T) {
	cases := []struct {
		b8, b9 byte
		want   string
	}{
		{0x10, 0xAC, "DEL"},
		{0x00, 0x00, ""}, // all-zero letters are out of A-Z range
	}
	for _, c := range cases {
		if got := decodeEDIDPNPID(c.b8, c.b9); got != c.want {
			t.Errorf("decodeEDIDPNPID(%#x, %#x) = %q, want %q", c.b8, c.b9, got, c.want)
		}
	}
}

func TestResolvePNPVendor(t *testing.T) {
	if got := resolvePNPVendor("DEL"); got != "Dell" {
		t.Errorf("resolvePNPVendor(DEL) = %q, want Dell", got)
	}
	if got := resolvePNPVendor("ZZZ"); got != "ZZZ" {
		t.Errorf("resolvePNPVendor(ZZZ) = %q, want ZZZ (raw fallback)", got)
	}
}
