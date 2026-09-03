package gp200

import (
	"bytes"
	"testing"
)

// TestDecodeUserControls verifies the EXP/CTRL assignment records of a real
// user preset decode to the expected wiring.
func TestDecodeUserControls(t *testing.T) {
	p, err := Unmarshal(mustRead(t, "user-fender-twin.prst"))
	if err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	// The device default wires EXP1 Mode A Para 1 to the VOL block (10).
	if p.Exp[0].Page != 0 || p.Exp[0].Item != 0 || p.Exp[0].Block != 10 {
		t.Fatalf("EXP1 Mode A Para 1 = %+v, want VOL block 10", p.Exp[0])
	}
	// All eight CTRL records must be present with their index.
	for i := range p.Ctrl {
		if p.Ctrl[i].Index != i {
			t.Fatalf("Ctrl[%d].Index = %d, want %d", i, p.Ctrl[i].Index, i)
		}
	}
}

// TestControlsRoundTrip builds a preset with custom footswitch and pedal
// assignments, re-decodes it, and checks the assignments and byte stability.
func TestControlsRoundTrip(t *testing.T) {
	p := New()
	p.PatchName = "Ctrl Test"
	// CTRL 1 toggles DST (block 2) and DLY (block 8); saved state on.
	p.Ctrl[0] = CtrlAssignment{Index: 0, BlockMask: 1<<2 | 1<<8, State: 1}
	// EXP1 Mode A Para 3 sweeps the DLY block's Mix (param 0), 0..50.
	p.Exp[2] = ExpAssignment{Page: 0, Item: 2, Block: 8, ParamIndex: 0, Min: 0, Max: 50}

	data, err := p.Marshal()
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	back, err := Unmarshal(data)
	if err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if back.Ctrl[0].BlockMask != 1<<2|1<<8 || back.Ctrl[0].State != 1 {
		t.Fatalf("Ctrl[0] = %+v", back.Ctrl[0])
	}
	if back.Exp[2].Block != 8 || back.Exp[2].ParamIndex != 0 || back.Exp[2].Max != 50 {
		t.Fatalf("Exp[2] = %+v", back.Exp[2])
	}
	// A second marshal must be byte-identical (synthetic round-trip stability).
	again, err := back.Marshal()
	if err != nil {
		t.Fatalf("second Marshal: %v", err)
	}
	if !bytes.Equal(again, data) {
		t.Fatal("second round-trip not byte-identical")
	}
}

// TestRoutingRoundTrip pins the signal-chain reorder bytes.
func TestRoutingRoundTrip(t *testing.T) {
	p := New()
	// Swap DLY (8) before RVB (9): a plausible reorder.
	p.Routing[8], p.Routing[9] = p.Routing[9], p.Routing[8]
	data, err := p.Marshal()
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	back, err := Unmarshal(data)
	if err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if back.Routing[8] != 9 || back.Routing[9] != 8 {
		t.Fatalf("routing not preserved: %v", back.Routing)
	}
}
