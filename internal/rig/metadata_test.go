package rig

import "testing"

func TestBuildUsesDeviceMetadataDefaults(t *testing.T) {
	b := newTestBuilder(t)
	file, err := b.Build(Spec{
		Name: "Meta",
		Blocks: []Block{
			{Type: "Amp", Params: map[string]any{"Type": "65 Black SR"}},
			{Type: "Cab", Params: map[string]any{"CabType": "1x12 Black Panel Lux"}},
		},
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if file.ProgNum != -1 {
		t.Fatalf("ProgNum = %d, want -1 (unassigned)", file.ProgNum)
	}
	if file.Color != 4 {
		t.Fatalf("Color = %d, want 4 (factory default)", file.Color)
	}
	if file.Readonly {
		t.Fatal("Readonly must be false")
	}
	if file.Author != "UserName" {
		t.Fatalf("Author = %q, want UserName", file.Author)
	}
	content, err := file.Decode()
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if content.Info.Version != "2.0.3" {
		t.Fatalf("version = %q, want 2.0.3", content.Info.Version)
	}
}

func TestBuildRejectsOutOfRangeColour(t *testing.T) {
	b := newTestBuilder(t)
	for _, color := range []int{-1, 10, 99} {
		_, err := b.Build(Spec{
			Name:  "Bad Colour",
			Color: color,
			Blocks: []Block{
				{Type: "Amp", Params: map[string]any{"Type": "65 Black SR"}},
				{Type: "Cab", Params: map[string]any{"CabType": "1x12 Black Panel Lux"}},
			},
		})
		if err == nil {
			t.Fatalf("expected error for colour %d", color)
		}
	}
}
