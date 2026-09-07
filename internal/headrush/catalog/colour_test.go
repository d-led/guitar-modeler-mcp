package catalog

import "testing"

func TestColourPalette(t *testing.T) {
	want := []string{
		"Blue", "Yellow", "Green", "Purple", "Red", "Dark Green", "Orange", "Light Blue", "Pink",
	}
	got := Colours()
	if len(got) != len(want) {
		t.Fatalf("Colours() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Colours() = %v, want %v", got, want)
		}
		if !ColourValid(want[i]) {
			t.Fatalf("ColourValid(%q) = false, want true", want[i])
		}
	}
	if ColourValid("Mauve") {
		t.Fatal("ColourValid(\"Mauve\") = true, want false")
	}
}
