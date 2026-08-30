package htmlreport

import "testing"

func TestChainStepsSerial(t *testing.T) {
	slots := []string{"Amp", "Cab", "Spring Reverb"}
	steps := chainSteps("", slots)
	if len(steps) != 3 {
		t.Fatalf("steps = %d, want 3", len(steps))
	}
	if steps[0].Slot != 1 || steps[1].Slot != 2 || steps[2].Slot != 3 {
		t.Errorf("serial slot numbers = %d,%d,%d, want 1,2,3", steps[0].Slot, steps[1].Slot, steps[2].Slot)
	}
}

func TestChainStepsSPSParallelJunction(t *testing.T) {
	slots := []string{
		"Green JRC-OD", "Amp", "Cab",
		"Tape Echo", "Empty Slot", "Empty Slot",
		"Eleven Reverb", "Empty Slot", "Empty Slot",
		"Tremolo", "Empty Slot",
	}
	steps := chainSteps("SPS-1", slots)
	// 3 prefix + 1 junction + 2 suffix.
	if len(steps) != 6 {
		t.Fatalf("steps = %d, want 6 (3 prefix + junction + 2 suffix)", len(steps))
	}
	junction := steps[3]
	if len(junction.Branches) != 2 {
		t.Fatalf("step 4 should be a two-branch junction: %+v", junction)
	}
	if got := junction.Branches[0].Steps[0]; got.Slot != 4 || got.Effect != "Tape Echo" {
		t.Errorf("branch A first step = %+v, want slot 4 Tape Echo", got)
	}
	if got := junction.Branches[1].Steps[0]; got.Slot != 7 || got.Effect != "Eleven Reverb" {
		t.Errorf("branch B first step = %+v, want slot 7 Eleven Reverb", got)
	}
	if steps[4].Slot != 10 || steps[4].Effect != "Tremolo" {
		t.Errorf("suffix first step = %+v, want slot 10 Tremolo", steps[4])
	}
}

func TestChainStepsPSParallelFromInput(t *testing.T) {
	slots := make([]string, 11)
	for i := range slots {
		slots[i] = "Empty Slot"
	}
	slots[0] = "Amp"
	slots[3] = "Amp 2"
	slots[8] = "Eleven Reverb"
	steps := chainSteps("PS-1", slots)
	if len(steps) != 4 {
		t.Fatalf("steps = %d, want 4 (junction + 3 suffix)", len(steps))
	}
	if len(steps[0].Branches) != 2 {
		t.Fatalf("step 1 should be the parallel junction: %+v", steps[0])
	}
	if steps[0].Branches[0].Steps[0].Slot != 1 || steps[0].Branches[0].Steps[0].Effect != "Amp" {
		t.Errorf("branch A first step = %+v, want slot 1 Amp", steps[0].Branches[0].Steps[0])
	}
	if steps[0].Branches[1].Steps[0].Slot != 4 || steps[0].Branches[1].Steps[0].Effect != "Amp 2" {
		t.Errorf("branch B first step = %+v, want slot 4 Amp 2", steps[0].Branches[1].Steps[0])
	}
	if steps[3].Slot != 11 {
		t.Errorf("last suffix slot = %d, want 11", steps[3].Slot)
	}
}
