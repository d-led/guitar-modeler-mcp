package rig

import (
	"bytes"
	"testing"
)

// TestSetContentNoOpPreservesSections guards the Load → Change → Save
// contract: a no-op round-trip keeps the raw FootSwitch/Pedal sections
// byte-for-byte, so only the part an editor touches changes.
func TestSetContentNoOpPreservesSections(t *testing.T) {
	b := newTestBuilder(t)
	file, err := b.Build(Spec{
		Name: "Round Trip",
		Blocks: []Block{
			{Type: "Chorus", Enabled: false},
			{Type: "Amp", Params: map[string]any{"Type": "65 Black SR"}},
			{Type: "Cab", Params: map[string]any{"CabType": "1x12 Black Panel Lux"}},
		},
		Footswitches: []Footswitch{{Module: "Chorus"}},
		Pedals:       []Pedal{{Module: "Chorus", Param: "Pedal"}},
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	content, err := file.Decode()
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	originalFoot := append([]byte(nil), content.FootSwitch...)
	originalPedal1 := append([]byte(nil), content.Pedal1...)
	originalPedal2 := append([]byte(nil), content.Pedal2...)

	if err := file.SetContent(content); err != nil {
		t.Fatalf("SetContent: %v", err)
	}
	re, err := file.Decode()
	if err != nil {
		t.Fatalf("Decode after SetContent: %v", err)
	}
	if !bytes.Equal(originalFoot, re.FootSwitch) {
		t.Error("FootSwitch section changed on a no-op round trip")
	}
	if !bytes.Equal(originalPedal1, re.Pedal1) {
		t.Error("Pedal1 section changed on a no-op round trip")
	}
	if !bytes.Equal(originalPedal2, re.Pedal2) {
		t.Error("Pedal2 section changed on a no-op round trip")
	}
}

// TestSetContentAppliesPatchEdits exercises the edit half of the round-trip:
// every numeric and boolean parameter of the Amp block is changed, saved, and
// re-read to confirm the edits persist.
func TestSetContentAppliesPatchEdits(t *testing.T) {
	b := newTestBuilder(t)
	file, err := b.Build(Spec{
		Name: "Edit Rig",
		Blocks: []Block{
			{Type: "Amp", Params: map[string]any{"Type": "65 Black SR"}},
			{Type: "Cab", Params: map[string]any{"CabType": "1x12 Black Panel Lux"}},
		},
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	content, err := file.Decode()
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}

	amp := content.Data.Patch.Children["Amp"]
	if amp == nil {
		t.Fatal("Amp node missing")
	}
	for _, item := range amp.Children {
		switch item.Type {
		case 0:
			v := 42.0
			item.Value = &v
		case 1, 3:
			on := true
			item.State = &on
		}
	}

	if err := file.SetContent(content); err != nil {
		t.Fatalf("SetContent: %v", err)
	}

	re, err := file.Decode()
	if err != nil {
		t.Fatalf("Decode after SetContent: %v", err)
	}
	reAmp := re.Data.Patch.Children["Amp"]
	for key, item := range reAmp.Children {
		switch item.Type {
		case 0:
			if item.Value == nil || *item.Value != 42 {
				t.Errorf("Amp %s = %v, want 42", key, item.Value)
			}
		case 1, 3:
			if item.State == nil || !*item.State {
				t.Errorf("Amp %s state = %v, want true", key, item.State)
			}
		}
	}
}

// TestSetContentTogglesModule exercises a block-level edit: bypassing a module
// persists through the round-trip.
func TestSetContentTogglesModule(t *testing.T) {
	b := newTestBuilder(t)
	file, err := b.Build(Spec{
		Name: "Toggle Edit",
		Blocks: []Block{
			{Type: "Chorus", Enabled: true},
			{Type: "Amp", Params: map[string]any{"Type": "65 Black SR"}},
			{Type: "Cab", Params: map[string]any{"CabType": "1x12 Black Panel Lux"}},
		},
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	content, err := file.Decode()
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	on := content.Data.Patch.Children["Chorus"].Children["On"]
	off := false
	on.State = &off

	if err := file.SetContent(content); err != nil {
		t.Fatalf("SetContent: %v", err)
	}
	re, err := file.Decode()
	if err != nil {
		t.Fatalf("Decode after SetContent: %v", err)
	}
	if state := re.Data.Patch.Children["Chorus"].Children["On"].State; state == nil || *state {
		t.Fatalf("Chorus On = %v, want false after edit", state)
	}
}
