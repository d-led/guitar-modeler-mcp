package thr

import (
	"strings"
	"testing"
)

// TestCaveatsFlagFlatAsAFRFRBypass covers the trap that reached the user: a
// FLAT amp is an FRFR bypass with no amp or speaker modelling, so a tone that
// must stand on the THR's own speaker does not belong on it.
func TestCaveatsFlagFlatAsAFRFRBypass(t *testing.T) {
	d := Default()
	spec, err := d.Resolve(Spec{Amp: "FLAT CLASSIC", Name: "x"})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	caveats := d.Caveats(spec)
	if !mentions(caveats, "FRFR bypass") {
		t.Fatalf("caveats = %v, want the FLAT/FRFR caveat", caveats)
	}
	// FLAT has no speaker modelling, so the cabinet selection is not a gap.
	if mentions(caveats, "No cabinet selected") {
		t.Errorf("caveats ask for a cabinet on a FLAT amp: %v", caveats)
	}
}

// TestCaveatsFlagAModellingAmpWithoutACabinet: the guitar amp families model
// their own speaker, so an empty cabinet sends an unmodelled amp to the unit's
// speaker. The BASS and ACOUSTIC groups ship with the cabinet bypassed, so they
// raise no such caveat (the user's own bass patch runs without one).
func TestCaveatsFlagAModellingAmpWithoutACabinet(t *testing.T) {
	d := Default()

	clean, err := d.Resolve(Spec{Amp: "CLEAN CLASSIC"})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if !mentions(d.Caveats(clean), "No cabinet selected") {
		t.Errorf("caveats = %v, want the missing-cabinet caveat", d.Caveats(clean))
	}

	withCab, err := d.Resolve(Spec{Amp: "CLEAN CLASSIC", Cab: "American 4x12"})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if mentions(d.Caveats(withCab), "No cabinet") {
		t.Errorf("caveats ask for a cabinet that was given: %v", d.Caveats(withCab))
	}

	bass, err := d.Resolve(Spec{Amp: "BASS BOUTIQUE"})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if mentions(d.Caveats(bass), "No cabinet") {
		t.Errorf("caveats ask for a cabinet on a bass amp: %v", d.Caveats(bass))
	}
}

// TestCaveatsStateWhereTheTHRLoudnessComesFrom: the .thrl6p has no output
// level, so a ported tone's loudness has to be re-derived from the amp's drive
// and master (plus the app compressor) — the user's port was quiet because the
// source amp's gain number was carried across. A bass tone gets the recipe
// verified on the unit so the numbers need not be guessed.
func TestCaveatsStateWhereTheTHRLoudnessComesFrom(t *testing.T) {
	d := Default()
	spec, err := d.Resolve(Spec{Amp: "BASS BOUTIQUE", AmpParams: AmpParams{Gain: Noon, Master: Noon}})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	caveats := d.Caveats(spec)
	if !mentions(caveats, "no output level") {
		t.Fatalf("caveats = %v, want the loudness caveat", caveats)
	}
	if !mentions(caveats, "Drive 60 / Master 100") {
		t.Errorf("bass loudness caveat does not carry the verified recipe: %v", caveats)
	}

	clean, err := d.Resolve(Spec{Amp: "CLEAN CLASSIC", Cab: "American 1x12", AmpParams: AmpParams{Gain: Noon}})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if mentions(d.Caveats(clean), "Drive 60 / Master 100") {
		t.Errorf("a guitar tone was given the bass recipe: %v", d.Caveats(clean))
	}
}

// TestCaveatsNameTheCompressorAsTheLoudnessThief: the app compressor squashes
// harder and drops the output as Sustain rises, which is what actually made the
// ported bass patch much quieter than the unit's own patches. A high Sustain is
// called out; a low-Sustain setting is not.
func TestCaveatsNameTheCompressorAsTheLoudnessThief(t *testing.T) {
	d := Default()

	hot, err := d.Resolve(Spec{Amp: "BASS CLASSIC", Compressor: true, CompParams: CompressorParams{Sustain: 65, Level: 65}})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	caveats := d.Caveats(hot)
	if !mentions(caveats, "Sustain 35 / Level 90") {
		t.Fatalf("caveats = %v, want the compressor caveat", caveats)
	}
	if !mentions(caveats, "net level loss") {
		t.Errorf("a Sustain above noon was not flagged: %v", caveats)
	}

	loud, err := d.Resolve(Spec{Amp: "BASS CLASSIC", Compressor: true, CompParams: CompressorParams{Sustain: 35, Level: 90}})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if mentions(d.Caveats(loud), "net level loss") {
		t.Errorf("the verified loud setting was flagged: %v", d.Caveats(loud))
	}

	// No compressor block, no compressor caveat: the legacy THR has no such
	// module in its chain at all.
	legacy, ok := ModelByName("thr10")
	if !ok {
		t.Fatal("thr10 not found")
	}
	legacySpec, err := legacy.Resolve(Spec{Amp: "Lead", Compressor: true})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if mentions(legacy.Caveats(legacySpec), "Dyna-Comp") {
		t.Errorf("a device without a compressor block got the compressor caveat: %v", legacy.Caveats(legacySpec))
	}
}

// TestCaveatsFlagAStarvedDrive: a Drive below noon leaves the amp model short
// of level as well as drive — the second half of the quiet-port story.
func TestCaveatsFlagAStarvedDrive(t *testing.T) {
	d := Default()

	starved, err := d.Resolve(Spec{Amp: "BASS CLASSIC", AmpParams: AmpParams{Gain: 43}})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if !mentions(d.Caveats(starved), "Drive 43") {
		t.Errorf("caveats = %v, want the starved-drive caveat", d.Caveats(starved))
	}

	for _, gain := range []int{Noon, 60, Unset} {
		spec, err := d.Resolve(Spec{Amp: "BASS CLASSIC", AmpParams: AmpParams{Gain: gain}})
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if mentions(d.Caveats(spec), "below noon") {
			t.Errorf("gain %d was flagged as starved: %v", gain, d.Caveats(spec))
		}
	}
}

// TestCaveatsAreSilentForLegacyModels: the legacy THR10/THR10C/THR10X write no
// preset file, so there is no hidden preset level to warn about.
func TestCaveatsAreSilentForLegacyModels(t *testing.T) {
	legacy, ok := ModelByName("thr10")
	if !ok {
		t.Fatal("thr10 not found")
	}
	spec, err := legacy.Resolve(Spec{Amp: "Lead", AmpParams: AmpParams{Gain: Noon}})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	if caveats := legacy.Caveats(spec); len(caveats) != 0 {
		t.Errorf("legacy caveats = %v, want none", caveats)
	}
}

func mentions(lines []string, want string) bool {
	return strings.Contains(strings.Join(lines, "\n"), want)
}
