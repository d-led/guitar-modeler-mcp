package gp200

import (
	"strings"
	"testing"
)

func TestNewInitializesBlockDefaults(t *testing.T) {
	p := New()
	// Unplaced blocks keep their module default, not a stray COMP (code 0).
	if name := EffectName(p.Blocks[4].EffectID); name != "Gate 1" {
		t.Fatalf("NR default = %q, want Gate 1", name)
	}
	if name := EffectName(p.Blocks[10].EffectID); name != "Volume" {
		t.Fatalf("VOL default = %q, want Volume", name)
	}
	if name := EffectName(p.Blocks[3].EffectID); name != "Tweedy" {
		t.Fatalf("AMP default = %q, want Tweedy", name)
	}
	// AMP, CAB and VOL start on; the rest start off.
	if !p.Blocks[3].Enabled || !p.Blocks[5].Enabled || !p.Blocks[10].Enabled {
		t.Fatal("AMP/CAB/VOL should start enabled")
	}
	if p.Blocks[0].Enabled || p.Blocks[4].Enabled {
		t.Fatal("PRE/NR should start disabled")
	}
}

func TestSetupCardListsSwitchableOffBlockParams(t *testing.T) {
	p := New()
	p.PatchName = "Switchable"
	// A wah that starts off and is toggled by CTRL 1.
	place(&p.Blocks[1], 1, 0x05000001, false, nil)
	p.Ctrl[0] = CtrlAssignment{Index: 0, BlockMask: 1 << 1}

	card := SetupCardHTML(Default(), p)

	// The off WAH block still lists its parameters and its footswitch, so the
	// user can dial it in before switching it on.
	mustContain(t, card, "WAH", "OFF", "V-Wah", "Range:", "Position:", "CTRL 1")
}

func TestSetupCardOmitsPlainOffBlockParams(t *testing.T) {
	p := New()
	p.PatchName = "Plain Off"
	// The NR block is off and not assigned to any footswitch: nothing to dial.
	card := SetupCardHTML(Default(), p)

	mustContain(t, card, "Gate 1")
	if strings.Contains(card, "Threshold:") {
		t.Fatalf("a plain off block should not list parameters:\n%s", card)
	}
}

func TestSetupCardFootswitchStateAndExpRange(t *testing.T) {
	p := New()
	p.PatchName = "With Footswitches"
	p.Ctrl[0] = CtrlAssignment{Index: 0, BlockMask: 1 << 1, State: 0} // CTRL 1 toggles WAH, off
	p.Ctrl[1] = CtrlAssignment{Index: 1, BlockMask: 1 << 7, State: 1} // CTRL 2 toggles MOD, on
	// EXP1 Mode B Para 1 sweeps the WAH Position from 15 to 85.
	p.Exp[3] = ExpAssignment{Page: 1, Item: 0, Block: 1, ParamIndex: 3, Min: 15, Max: 85}

	card := SetupCardHTML(Default(), p)
	mustContain(t, card, "WAH", "MOD", "EXP1 B P1", "Position (15–85)")
	// The CTRL boxes carry their saved toggle position: CTRL 1 off, CTRL 2 on.
	mustContain(t, card, "<div class=\"btn off\">", "<div class=\"btn on\">")
}

func TestSetupCardShowsSwitchOptionsByName(t *testing.T) {
	p := New()
	place(&p.Blocks[8], 8, 184549377, true, nil) // Analog delay has Sync/Trail switches
	card := SetupCardHTML(Default(), p)
	mustContain(t, card, "Sync: OFF", "Trail: OFF")
	if strings.Contains(card, "Sync: 0") {
		t.Fatalf("switch should render by option name, not raw index:\n%s", card)
	}
}

func TestSetupCardShowsUnits(t *testing.T) {
	p := New()
	place(&p.Blocks[7], 7, 67108864, true, map[string]float32{"Rate": 0.5, "Depth": 30}) // A-Chorus
	place(&p.Blocks[8], 8, 184549377, true, map[string]float32{"Time": 400})             // Analog delay
	card := SetupCardHTML(Default(), p)
	mustContain(t, card, "Rate: 0.5", "Time: 400", "Depth: 30", `<span class="unit">Hz</span>`, `<span class="unit">ms</span>`)
}

func TestSetupCardShowsUnknownUnitQuestion(t *testing.T) {
	p := New()
	place(&p.Blocks[9], 9, 201326593, true, nil) // Hall reverb: Pre Delay is 0..100, unit ambiguous
	card := SetupCardHTML(Default(), p)
	mustContain(t, card, "Pre Delay", `<span class="unit">?</span>`)
}

func mustContain(t *testing.T, text string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(text, want) {
			t.Fatalf("output missing %q:\n%s", want, text)
		}
	}
}
