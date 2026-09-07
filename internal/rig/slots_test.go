package rig

import "testing"

func TestChainSerialSlotsRespectPins(t *testing.T) {
	c := chain{
		routing: RoutingSerial,
		serial: []Block{
			{Type: "DynIII Comp"},
			{Type: "Amp"},
			{Type: "Cab"},
			{Type: "Dyn Delay"},
		},
		pins: map[int]Block{1: {Type: "Volume"}},
	}

	want := []string{
		"Volume", "DynIII Comp", "Amp", "Cab", "Dyn Delay",
		"Empty Slot", "Empty Slot", "Empty Slot", "Empty Slot", "Empty Slot", "Empty Slot",
	}
	assertSlots(t, c.slots(), want)
}

func TestChainSerialSlotsLeaveGapAtPinnedSlot(t *testing.T) {
	c := chain{
		routing: RoutingSerial,
		serial: []Block{
			{Type: "DynIII Comp"},
			{Type: "Amp"},
			{Type: "Cab"},
			{Type: "Dyn Delay"},
		},
		pins: map[int]Block{11: {Type: "Volume"}},
	}

	want := []string{
		"DynIII Comp", "Amp", "Cab", "Dyn Delay",
		"Empty Slot", "Empty Slot", "Empty Slot", "Empty Slot", "Empty Slot", "Empty Slot",
		"Volume",
	}
	assertSlots(t, c.slots(), want)
}

func TestChainSerialSlotsWithoutPinsIsSequential(t *testing.T) {
	c := chain{
		routing: RoutingSerial,
		serial:  []Block{{Type: "Amp"}, {Type: "Cab"}},
	}

	want := []string{"Amp", "Cab", "Empty Slot", "Empty Slot", "Empty Slot", "Empty Slot", "Empty Slot", "Empty Slot", "Empty Slot", "Empty Slot", "Empty Slot"}
	assertSlots(t, c.slots(), want)
}

func assertSlots(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("slots = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("slot %d = %q, want %q (slots = %v)", i+1, got[i], want[i], got)
		}
	}
}
