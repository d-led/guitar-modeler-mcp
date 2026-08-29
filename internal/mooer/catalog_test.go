package mooer

import "testing"

// ge150proModel returns the canonical GE150 Pro Li model, whose tables were
// reverse-engineered from the device backup.
func ge150proModel(t *testing.T) Model {
	t.Helper()
	m, ok := ModelByName("ge150pro")
	if !ok {
		t.Fatal("ge150pro not registered")
	}
	return m
}

func TestEffectNameForAmpAndCab(t *testing.T) {
	m := ge150proModel(t)
	if got := m.EffectName("amp", 2); got != "Brit 800" {
		t.Fatalf("amp[2] = %q, want Brit 800", got)
	}
	if got := m.EffectName("cab", 17); got != "4x12 V30" {
		t.Fatalf("cab[17] = %q, want 4x12 V30", got)
	}
	if got := m.EffectName("amp", 200); got != "" {
		t.Fatalf("amp[200] = %q, want empty", got)
	}
}

func TestEffectNameForFixedModules(t *testing.T) {
	m := ge150proModel(t)
	if got := m.EffectName("ns", 0); got != "Noise Gate" {
		t.Fatalf("ns = %q, want Noise Gate", got)
	}
	if got := m.EffectName("eq", 0); got != "EQ" {
		t.Fatalf("eq = %q, want EQ", got)
	}
}

func TestEffectIndexRoundTrip(t *testing.T) {
	m := ge150proModel(t)
	for _, tc := range []struct {
		module string
		name   string
		want   uint8
	}{
		{"od", "Tube Screamer", 11},
		{"delay", "Tape", 2},
		{"reverb", "Hall", 1},
		{"mod", "Chorus", 0},
	} {
		index, ok := m.EffectIndex(tc.module, tc.name)
		if !ok {
			t.Fatalf("EffectIndex(%q, %q) not found", tc.module, tc.name)
		}
		if index != tc.want {
			t.Fatalf("EffectIndex(%q, %q) = %d, want %d", tc.module, tc.name, index, tc.want)
		}
		if got := m.EffectName(tc.module, index); got != tc.name {
			t.Fatalf("EffectName(%q, %d) = %q, want %q", tc.module, index, got, tc.name)
		}
	}
}

func TestAmpAndCabIndex(t *testing.T) {
	m := ge150proModel(t)
	if index, ok := m.AmpIndex("Twin Reverb"); !ok || index != 21 {
		t.Fatalf("AmpIndex(Twin Reverb) = %d, %v; want 21, true", index, ok)
	}
	if index, ok := m.CabIndex("4x12 Green"); !ok || index != 18 {
		t.Fatalf("CabIndex(4x12 Green) = %d, %v; want 18, true", index, ok)
	}
	if _, ok := m.AmpIndex("Not An Amp"); ok {
		t.Fatal("AmpIndex found a name that does not exist")
	}
}

func TestModuleCounts(t *testing.T) {
	m := ge150proModel(t)
	if len(m.Amps) != 55 {
		t.Fatalf("Amps has %d entries, want 55", len(m.Amps))
	}
	if len(m.Cabs) != 26 {
		t.Fatalf("Cabs has %d entries, want 26", len(m.Cabs))
	}
	for module, want := range map[string]int{
		"fx": 13, "od": 12, "mod": 13, "delay": 9, "reverb": 9,
	} {
		if got := len(m.Effects[module]); got != want {
			t.Fatalf("Effects[%q] has %d entries, want %d", module, got, want)
		}
	}
}
