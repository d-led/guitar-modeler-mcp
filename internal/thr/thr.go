// Package thr is the device-specific backend for the Yamaha THR desktop amp
// series. It has no publicly documented preset file format, so the output is
// a printable setup card. The THR-II amp selector and effect lists come from
// Yamaha's official specs and manual; the "inspired by" real-amplifier names
// and the legacy THR10/THR10C/THR10X patches are community-sourced.
package thr

import (
	"fmt"
	"strings"
)

// Item is one selectable effect, with the real hardware it emulates (empty
// when not documented).
type Item struct {
	Name       string `json:"name"`
	InspiredBy string `json:"inspired_by,omitempty"`
}

// AmpCell is one position of the amp selector. On the THR-II, Type is the
// selector group (CLEAN..FLAT) and Mode the variant (CLASSIC/BOUTIQUE/MODERN).
// On the legacy models the patches are flat, so Type and Mode are empty.
type AmpCell struct {
	Name        string `json:"name"`
	Type        string `json:"type,omitempty"`
	Mode        string `json:"mode,omitempty"`
	Description string `json:"description,omitempty"`
	InspiredBy  string `json:"inspired_by,omitempty"`
}

// Device describes one Yamaha THR model.
type Device struct {
	Name         string
	Display      string
	FileExchange bool
	FileExt      string
	// Note is shown on the setup card when the catalog is known to be partial.
	Note       string
	Chain      []string
	AmpTypes   []string
	AmpModes   []string
	Amps       []AmpCell
	Modulation []Item
	EchoRev    []Item
}

// Effect lists shared across the THR line (the two physical knobs).
var (
	thrModulation = items(
		[2]string{"CHORUS", ""},
		[2]string{"FLANGER", ""},
		[2]string{"PHASER", ""},
		[2]string{"TREMOLO", ""},
	)
	thrEchoRev = items(
		[2]string{"ECHO", ""},
		[2]string{"ECHO/REV", ""},
		[2]string{"SPRING REVERB", ""},
		[2]string{"HALL REVERB", ""},
	)
)

func items(pairs ...[2]string) []Item {
	out := make([]Item, len(pairs))
	for i, p := range pairs {
		out[i] = Item{Name: p[0], InspiredBy: p[1]}
	}
	return out
}

// ampGrid builds THR-II amp cells: name, type, mode, description, inspired-by.
func ampGrid(rows ...[5]string) []AmpCell {
	out := make([]AmpCell, len(rows))
	for i, r := range rows {
		out[i] = AmpCell{Name: r[0], Type: r[1], Mode: r[2], Description: r[3], InspiredBy: r[4]}
	}
	return out
}

func ampItems(rows ...[2]string) []AmpCell {
	out := make([]AmpCell, len(rows))
	for i, r := range rows {
		out[i] = AmpCell{Name: r[0], InspiredBy: r[1]}
	}
	return out
}

// thrII is the current generation (THR10II, THR30II, THR10II Wireless). The
// amp selector has eight groups, each with a CLASSIC/BOUTIQUE/MODERN variant
// (FLAT has no variant).
func thrII() Device {
	return Device{
		Name:         "thr",
		Display:      "Yamaha THR-II",
		FileExchange: false,
		Chain:        []string{"COMPRESSOR", "NOISE GATE", "AMP", "MOD", "ECHO/REV"},
		AmpTypes:     []string{"CLEAN", "CRUNCH", "LEAD", "HI GAIN", "SPECIAL", "BASS", "ACOUSTIC", "FLAT"},
		AmpModes:     []string{"CLASSIC", "BOUTIQUE", "MODERN"},
		Amps: ampGrid(
			[5]string{"CLEAN CLASSIC", "CLEAN", "CLASSIC", "A low-gain preamp for sparkling American-style cleans, with 6L6 tubes in the output stage for brightness and a strong midrange.", "Fender Twin Reverb / Deluxe Reverb"},
			[5]string{"CLEAN BOUTIQUE", "CLEAN", "BOUTIQUE", "A low-watt EL34 design; preamp gain thickens the cleans into bluesy overdrive.", "Matchless DC30"},
			[5]string{"CLEAN MODERN", "CLEAN", "MODERN", "A boutique, low-watt EL84 design that adds fullness and sustain; a great match for neck pickups.", "Dr. Z Carmen Ghia"},
			[5]string{"CRUNCH CLASSIC", "CRUNCH", "CLASSIC", "EL84 power tubes in a true Class-A configuration with a highly responsive EQ; inspired by British chime.", "Vox AC30"},
			[5]string{"CRUNCH BOUTIQUE", "CRUNCH", "BOUTIQUE", "A simple circuit with a single 12AX7 and EL84; full, no-frills and responsive to picking dynamics.", "Marshall Bluesbreaker / Plexi"},
			[5]string{"CRUNCH MODERN", "CRUNCH", "MODERN", "A mid-volume boutique design with 6550 power tubes; tight bass and a singing sustain.", "Dumble Amp / Friedman style"},
			[5]string{"LEAD CLASSIC", "LEAD", "CLASSIC", "A low-gain preamp with an EL34 power section that breaks into classic British overdrive.", "Marshall Plexi / JCM800"},
			[5]string{"LEAD BOUTIQUE", "LEAD", "BOUTIQUE", "The Classic/Lead circuit modified for extra gain, a darker tone and scooped mids.", "Marshall 1987X Plexi"},
			[5]string{"LEAD MODERN", "LEAD", "MODERN", "A high-gain design with 12AX7s into EL34s; the tone that defined 1980s hard rock and metal.", "Soldano SLO-100"},
			[5]string{"HI GAIN CLASSIC", "HI GAIN", "CLASSIC", "Powerful modern distortion that fills out as the high-gain preamp is pushed.", "Mesa/Boogie Dual Rectifier"},
			[5]string{"HI GAIN BOUTIQUE", "HI GAIN", "BOUTIQUE", "ECC83s into 6L6s for high gain with a highly responsive EQ; inspired by German engineering.", "EVH 5150-III (Channel 2)"},
			[5]string{"HI GAIN MODERN", "HI GAIN", "MODERN", "A boosted Classic/Special amp with even more gain for aggressive rhythms or searing leads.", "EVH 5150-III (Channel 3)"},
			[5]string{"SPECIAL CLASSIC", "SPECIAL", "CLASSIC", "12AX7 and 6L6 tubes in pursuit of the 'Brown' sound; classic-rock crunch to saturated rhythm.", "Modified Marshall Plexi (Brown Sound)"},
			[5]string{"SPECIAL BOUTIQUE", "SPECIAL", "BOUTIQUE", "Four 12AX7 preamp tubes into 6L6 output tubes; tight, fast tracking for crushing gain.", "Brown Sound / modified EVH"},
			[5]string{"SPECIAL MODERN", "SPECIAL", "MODERN", "A classic overdrive circuit before the preamp tightens the lows and adds gain; ideal for extended-range guitars.", "EVH 5150 Stealth"},
			[5]string{"BASS CLASSIC", "BASS", "CLASSIC", "Woody, vintage tone with late breakup.", "Ampeg SVT"},
			[5]string{"BASS BOUTIQUE", "BASS", "BOUTIQUE", "Full, modern tone that breaks into a fuzz-like overdrive when pushed hard.", "Mesa/Boogie Subway Series"},
			[5]string{"BASS MODERN", "BASS", "MODERN", "Vintage voicing with early breakup; works well with bass or guitar.", "Markbass Little Marcus / Eden Terra Nova"},
			[5]string{"ACOUSTIC CLASSIC", "ACOUSTIC", "CLASSIC", "Models the response of a boutique condenser microphone.", "Condenser studio microphone"},
			[5]string{"ACOUSTIC BOUTIQUE", "ACOUSTIC", "BOUTIQUE", "Models the response of a boutique tube microphone.", "Tube microphone"},
			[5]string{"ACOUSTIC MODERN", "ACOUSTIC", "MODERN", "Models the response of a boutique dynamic microphone.", "Dynamic instrument microphone"},
			[5]string{"FLAT CLASSIC", "FLAT", "CLASSIC", "A neutral tone with no amp or speaker modeling.", "FRFR bypass"},
			[5]string{"FLAT BOUTIQUE", "FLAT", "BOUTIQUE", "A neutral tone with no amp or speaker modeling and a slight bass boost.", "FRFR bypass (bass boost)"},
			[5]string{"FLAT MODERN", "FLAT", "MODERN", "A neutral tone with no amp or speaker modeling and a slight mid scoop.", "FRFR bypass (mid scoop)"},
		),
		Modulation: thrModulation,
		EchoRev:    thrEchoRev,
	}
}

// thr10, thr10c and thr10x are the legacy models. Their amp lists below are
// partial: they hold only the patches documented in the community reference,
// not the full factory set.
func thr10() Device {
	return Device{
		Name:         "thr10",
		Display:      "Yamaha THR10",
		FileExchange: false,
		Note:         "Partial amp list (community reference); not the full factory set.",
		Chain:        []string{"AMP", "MOD", "ECHO/REV"},
		Amps: ampItems(
			[2]string{"Lead", "Marshall Plexi"},
			[2]string{"Modern", "Mesa/Boogie Rectifier-style high gain"},
			[2]string{"Brit Hi", "Marshall JCM800"},
		),
		Modulation: thrModulation,
		EchoRev:    thrEchoRev,
	}
}

func thr10c() Device {
	return Device{
		Name:         "thr10c",
		Display:      "Yamaha THR10C",
		FileExchange: false,
		Note:         "Partial amp list (community reference); not the full factory set.",
		Chain:        []string{"AMP", "MOD", "ECHO/REV"},
		Amps: ampItems(
			[2]string{"Deluxe", "Fender Twin Reverb / Deluxe Reverb"},
			[2]string{"Class A", "Matchless DC30"},
			[2]string{"US Blues", "Fender Blues Jr."},
			[2]string{"Mini", "Dr. Z Mini Z"},
		),
		Modulation: thrModulation,
		EchoRev:    thrEchoRev,
	}
}

func thr10x() Device {
	return Device{
		Name:         "thr10x",
		Display:      "Yamaha THR10X",
		FileExchange: false,
		Note:         "Partial amp list (community reference); not the full factory set.",
		Chain:        []string{"AMP", "MOD", "ECHO/REV"},
		Amps: ampItems(
			[2]string{"Brown 1", "Early Van Halen Marshall (Brown Sound)"},
			[2]string{"Southern Hi", "Dimebag Darrell-style high gain"},
			[2]string{"Brown 2", "EVH 5150 (later Van Halen)"},
		),
		Modulation: thrModulation,
		EchoRev:    thrEchoRev,
	}
}

// models is the registry of supported THR devices, in display order.
var models = []Device{thrII(), thr10(), thr10c(), thr10x()}

// Models returns the supported THR devices.
func Models() []Device {
	return models
}

// ModelByName returns the device with the given identifier, matching
// case-insensitively against the stable name or the display name.
func ModelByName(name string) (Device, bool) {
	q := strings.ToLower(strings.TrimSpace(name))
	for _, m := range models {
		if strings.ToLower(m.Name) == q || strings.ToLower(m.Display) == q {
			return m, true
		}
	}
	return Device{}, false
}

// Default returns the canonical THR device (the THR-II).
func Default() Device {
	return thrII()
}

// Resolve canonicalises a Spec's selections to on-device names. The amp field
// matches a full cell name ("CLEAN CLASSIC"), a selector group alone (which
// resolves to the CLASSIC variant), or a substring of the "inspired by"
// description; it is required. The effect fields may be left empty ("off").
func (d Device) Resolve(s Spec) (Spec, error) {
	cell, err := d.resolveAmp(s.Amp)
	if err != nil {
		return s, err
	}
	s.Amp = cell.Name

	if s.Mod, err = resolveItem(d.Modulation, s.Mod, "modulation"); err != nil {
		return s, err
	}
	if s.EchoRev, err = resolveItem(d.EchoRev, s.EchoRev, "echo/rev"); err != nil {
		return s, err
	}
	return s, nil
}

func (d Device) resolveAmp(query string) (AmpCell, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		return AmpCell{}, fmt.Errorf("an amp is required")
	}
	qn := norm(q)

	// Full cell name ("clean classic").
	for _, c := range d.Amps {
		if norm(c.Name) == qn {
			return c, nil
		}
	}
	// Selector group alone ("clean" → the CLASSIC variant).
	for _, c := range d.Amps {
		if c.Type != "" && strings.EqualFold(c.Type, q) && c.Mode == d.defaultMode() {
			return c, nil
		}
	}
	// "Inspired by" substring ("twin reverb").
	for _, c := range d.Amps {
		if c.InspiredBy != "" && strings.Contains(norm(c.InspiredBy), qn) {
			return c, nil
		}
	}
	return AmpCell{}, fmt.Errorf("no amp matches %q (see thr_catalog_list_amps)", query)
}

func (d Device) defaultMode() string {
	if len(d.AmpModes) > 0 {
		return d.AmpModes[0]
	}
	return ""
}

// ampCell returns the amp cell for a canonical name.
func (d Device) ampCell(name string) (AmpCell, bool) {
	for _, c := range d.Amps {
		if strings.EqualFold(c.Name, name) {
			return c, true
		}
	}
	return AmpCell{}, false
}

func resolveItem(items []Item, query, label string) (string, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		return "", nil
	}
	for _, it := range items {
		if strings.EqualFold(it.Name, q) {
			return it.Name, nil
		}
	}
	for _, it := range items {
		if it.InspiredBy != "" && strings.Contains(norm(it.InspiredBy), norm(q)) {
			return it.Name, nil
		}
	}
	return "", fmt.Errorf("no %s matches %q", label, query)
}

func norm(s string) string {
	s = strings.ReplaceAll(s, "·", " ")
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(s))), " ")
}
