package thr

import (
	"strconv"
	"strings"
)

// moduleCompressor is the app-only compressor's module name in the chain.
const moduleCompressor = "COMPRESSOR"

// Caveats returns the dial-in problems a THR tone carries that the .thrl6p
// cannot express and that the card alone does not call out. They are the
// traps found on the unit, so the design tool passes them on with every tone.
//
// The list is empty for a tone that is not audibly affected.
func (d Device) Caveats(s Spec) []string {
	var out []string
	if d.ampType(s.Amp) == ampTypeFlat {
		out = append(out, "FLAT is FRFR bypass: no amp and no speaker modelling. It is for feeding an "+
			"external preamp; through the THR's own speaker it is hollow and much quieter than a modelling "+
			"amp. For bass use the BASS group (BASS CLASSIC/BOUTIQUE/MODERN).")
	}
	if d.ampModelsSpeaker(s.Amp) && strings.TrimSpace(s.Cab) == "" {
		out = append(out, "No cabinet selected: the amp reaches the THR's speaker unmodelled. Choose one in "+
			"the THR Remote app (American 4x12 has the most low-end authority).")
	}
	if d.FileExchange {
		out = append(out, loudnessCaveat(d.ampType(s.Amp)))
	}
	if d.hasModule(moduleCompressor) && s.Compressor {
		out = append(out, compressorCaveat(s.CompParams))
	}
	if starvedDrive(s.AmpParams.Gain) {
		out = append(out, starvedDriveCaveat(s.AmpParams.Gain))
	}
	return out
}

// loudnessCaveat states where a THR tone's loudness actually comes from: the
// preset file has no output level, so it lives in the amp's Drive and Master
// plus the compressor's Level. A bass tone also gets the recipe verified on the
// unit, because a copied full-range gain is how a ported bass patch came out
// quieter than the unit's own patches.
func loudnessCaveat(ampType string) string {
	base := "The .thrl6p has no output level: the tone's loudness is the amp's Drive and Master plus the " +
		"compressor's Level, and nothing else in the file can compensate (the panel MASTER VOLUME is the " +
		"same for every preset, so it cannot explain a difference between them). Derive these for the THR " +
		"instead of copying the source's values."
	if ampType != ampTypeBass {
		return base
	}
	return base + " The verified loud bass recipe is BASS CLASSIC with Drive 60 / Master 100, the compressor " +
		"at Sustain 35 / Level 90, and the American 4x12 cabinet."
}

// compressorCaveat names the app compressor as the usual loudness thief: the
// RedComp is Dyna-Comp style, so Sustain squashes harder AND lowers the output,
// which Level then has to win back.
func compressorCaveat(p CompressorParams) string {
	caveat := "The app compressor is Dyna-Comp style: Sustain squashes harder AND lowers the output, so " +
		"reaching for Sustain to get punch turns the patch down (it reads as squashed, not driven). Keep " +
		"Sustain low and win the level back with Level — the verified setting is Sustain 35 / Level 90."
	if p.Sustain > Noon {
		caveat += " This tone runs Sustain " + strconv.Itoa(p.Sustain) + ": that setting is a net level loss."
	}
	return caveat
}

// starvedDrive reports whether a Drive value leaves the amp model short of level
// as well as drive. An unset knob keeps the neutral noon default.
func starvedDrive(gain int) bool {
	return gain >= 0 && gain < Noon
}

// starvedDriveCaveat points a low Drive at the controls that should carry a
// clean tone's level instead.
func starvedDriveCaveat(drive int) string {
	return "Drive " + strconv.Itoa(drive) + " is below noon: the amp model is short of level as well as " +
		"drive, which is the second way a ported THR tone ends up quieter than the unit's own patches. A " +
		"clean tone keeps the Drive and takes its level from Master and the compressor's Level — the " +
		"verified loud bass setting is Drive 60 / Master 100."
}

// amp type identifiers, as the THR-II amp selector names them.
const (
	ampTypeBass     = "BASS"
	ampTypeAcoustic = "ACOUSTIC"
	ampTypeFlat     = "FLAT"
)

// ampType returns the selector group of a resolved amp name ("" on the legacy
// models, whose amp list carries no groups).
func (d Device) ampType(name string) string {
	cell, ok := d.ampCell(name)
	if !ok {
		return ""
	}
	return strings.ToUpper(cell.Type)
}

// ampModelsSpeaker reports whether the amp models its own speaker, so the
// cabinet selection is part of the tone. The BASS, ACOUSTIC and FLAT groups do
// not (BASS and ACOUSTIC ship with the cabinet bypassed, and FLAT is FRFR).
func (d Device) ampModelsSpeaker(name string) bool {
	switch d.ampType(name) {
	case "", ampTypeBass, ampTypeAcoustic, ampTypeFlat:
		return false
	default:
		return true
	}
}

// hasModule reports whether the device's chain carries a module at all.
func (d Device) hasModule(name string) bool {
	for _, m := range d.Chain {
		if m == name {
			return true
		}
	}
	return false
}
