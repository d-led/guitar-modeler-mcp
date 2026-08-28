package rig

import "testing"

func TestApplyParamsOverridesDefaults(t *testing.T) {
	node := newNode("Gain", "Tone", "On", "Mode")
	node.Children["Gain"] = num(50)
	node.Children["Tone"] = num(50)
	node.Children["On"] = boolean(true)
	node.Children["Mode"] = str("Clean")

	applyParams(node.Children, map[string]any{
		"Gain": 80.0,
		"On":   false,
		"Mode": "Drive",
	})

	if *node.Children["Gain"].Value != 80 {
		t.Fatalf("Gain = %v, want 80", *node.Children["Gain"].Value)
	}
	if *node.Children["On"].State != false {
		t.Fatal("On should be false after override")
	}
	if *node.Children["Mode"].Str != "Drive" {
		t.Fatalf("Mode = %v, want Drive", *node.Children["Mode"].Str)
	}
}

func TestApplyParamsAddsUnknownKeyWithInferredType(t *testing.T) {
	node := newNode("Gain")
	node.Children["Gain"] = num(50)

	applyParams(node.Children, map[string]any{
		"Level":  75.0,
		"Bright": true,
	})

	if *node.Children["Level"].Value != 75 {
		t.Fatalf("Level = %v, want 75", *node.Children["Level"].Value)
	}
	if node.Children["Level"].Type != 0 {
		t.Fatalf("Level type = %d, want 0", node.Children["Level"].Type)
	}
	if *node.Children["Bright"].State != true {
		t.Fatal("Bright should be true")
	}
}
