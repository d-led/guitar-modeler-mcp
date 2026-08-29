package catalog

import "testing"

func TestCabsMatching(t *testing.T) {
	c := New()
	if got := c.CabsMatching("Green"); len(got) != 2 {
		t.Fatalf("CabsMatching(Green) = %d cabs, want 2", len(got))
	}
	if got := c.CabsMatching(""); len(got) != len(c.Cabs()) {
		t.Fatalf("CabsMatching(\"\") = %d cabs, want all %d", len(got), len(c.Cabs()))
	}
	if got := c.CabsMatching("no-such-cab"); len(got) != 0 {
		t.Fatalf("CabsMatching(no-such-cab) = %d, want 0", len(got))
	}
}

func TestMicsMatching(t *testing.T) {
	c := New()
	if got := c.MicsMatching("Shure"); len(got) != 2 {
		t.Fatalf("MicsMatching(Shure) = %d mics, want 2", len(got))
	}
	if got := c.MicsMatching(""); len(got) != len(c.Mics()) {
		t.Fatalf("MicsMatching(\"\") = %d mics, want all %d", len(got), len(c.Mics()))
	}
}
