package hardware

// pnpVendors maps 3-letter EDID/PNP manufacturer IDs to human-readable
// vendor names. EDID only encodes the 3-letter code (e.g. "DEL"), not a
// readable name, so this is a manually maintained, best-effort table; new
// codes should be added as they're seen in the wild.
var pnpVendors = map[string]string{
	"DEL": "Dell",
	"SAM": "Samsung",
	"SDC": "Samsung",
	"LGD": "LG Display",
	"GSM": "LG Electronics",
	"AUS": "ASUS",
	"ACI": "ASUS",
	"BNQ": "BenQ",
	"ACR": "Acer",
	"AOC": "AOC",
	"VSC": "ViewSonic",
	"PHL": "Philips",
	"SNY": "Sony",
	"APP": "Apple",
	"HWP": "HP",
	"MSI": "MSI",
	"GBT": "Gigabyte",
	"NEC": "NEC",
	"ENC": "EIZO",
	"IVM": "Iiyama",
	"HSD": "HannStar",
	"CMN": "Chimei Innolux",
}

// resolvePNPVendor returns the human-readable vendor name for a 3-letter
// PNP ID, or the raw ID itself if unmapped.
func resolvePNPVendor(pnpID string) string {
	if name, ok := pnpVendors[pnpID]; ok {
		return name
	}
	return pnpID
}
