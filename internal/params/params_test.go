package params

import (
	"testing"

	"github.com/d-led/guitar-modeler-mcp/internal/catalog"
)

func TestDescribeAmpUsesCatalogModels(t *testing.T) {
	cat := catalog.New()
	spec, err := Describe(cat, "Amp")
	if err != nil {
		t.Fatalf("Describe Amp: %v", err)
	}
	values := spec["Type"].Values
	if len(values) != len(cat.Amps()) {
		t.Fatalf("Amp Type has %d values, want %d catalog models", len(values), len(cat.Amps()))
	}
	// The editor's list is missing this model; the catalog must supply it.
	for _, v := range values {
		if v == "92 Treadplate Modern" {
			return
		}
	}
	t.Fatal("catalog Amp list missing 92 Treadplate Modern")
}

func TestDescribeCab(t *testing.T) {
	cat := catalog.New()
	spec, err := Describe(cat, "Cab")
	if err != nil {
		t.Fatalf("Describe Cab: %v", err)
	}
	if len(spec["CabType"].Values) != len(cat.Cabs()) {
		t.Fatalf("CabType values = %d, want %d", len(spec["CabType"].Values), len(cat.Cabs()))
	}
	if len(spec["MicType"].Values) != len(cat.Mics()) {
		t.Fatalf("MicType values = %d, want %d", len(spec["MicType"].Values), len(cat.Mics()))
	}
}

func TestDescribeIsCaseInsensitive(t *testing.T) {
	cat := catalog.New()
	if _, err := Describe(cat, "tape echo"); err != nil {
		t.Fatalf("case-insensitive lookup failed: %v", err)
	}
}

func TestDescribeUnknownModule(t *testing.T) {
	cat := catalog.New()
	if _, err := Describe(cat, "Bogus"); err == nil {
		t.Fatal("expected error for unknown module")
	}
}
