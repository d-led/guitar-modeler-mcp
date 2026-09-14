package mooer

import (
	"strings"
	"testing"
)

// A bank is the device's own way of holding a song's variations, so the counts
// behind it are facts worth pinning: the GE100 Pro's editor lists 150 presets as
// 50 banks of three (01A..50C), and the GE150 Max / Max Li manual addresses 200
// as 50 banks of four (01A..50D).
func TestBankAddressedDevices(t *testing.T) {
	for _, tc := range []struct {
		model            string
		banks            int
		positionsPerBank int
		first            string
		last             string
	}{
		{model: "ge100pro", banks: 50, positionsPerBank: 3, first: "01A", last: "50C"},
		{model: "ge150pro", banks: 50, positionsPerBank: 4, first: "01A", last: "50D"},
	} {
		m, ok := ModelByName(tc.model)
		if !ok {
			t.Fatalf("%s is not registered", tc.model)
		}
		if m.Banks != tc.banks || m.PositionsPerBank != tc.positionsPerBank {
			t.Fatalf("%s holds %d banks of %d, want %d banks of %d",
				tc.model, m.Banks, m.PositionsPerBank, tc.banks, tc.positionsPerBank)
		}
		for _, want := range []struct {
			bank     int
			position int
			address  string
		}{
			{1, 0, tc.first},
			{tc.banks, tc.positionsPerBank - 1, tc.last},
		} {
			address, err := m.PresetAddress(want.bank, want.position)
			if err != nil {
				t.Fatalf("%s PresetAddress(%d, %d): %v", tc.model, want.bank, want.position, err)
			}
			if address != want.address {
				t.Fatalf("%s PresetAddress(%d, %d) = %q, want %q",
					tc.model, want.bank, want.position, address, want.address)
			}
		}
	}
}

// A device with no bank addressing has nowhere to file a bank design, so the
// addressing helper says so rather than inventing an address.
func TestDevicesWithoutBankAddressingRefuseAnAddress(t *testing.T) {
	for _, name := range []string{"ge200", "ge150"} {
		m, ok := ModelByName(name)
		if !ok {
			t.Fatalf("%s is not registered", name)
		}
		if m.BankAddressed() {
			t.Fatalf("%s should have no bank addressing, got %d banks of %d", name, m.Banks, m.PositionsPerBank)
		}
		if _, err := m.PresetAddress(1, 0); err == nil {
			t.Fatalf("%s accepted an address although it has no banks", name)
		} else if !strings.Contains(err.Error(), "has no bank addressing") {
			t.Fatalf("%s error = %q, want it to say the device has no bank addressing", name, err)
		}
	}
}

// An address outside the device's banks is refused rather than rounded onto
// whatever preset sits nearest.
func TestPresetAddressRefusesAddressesTheDeviceHasNotGot(t *testing.T) {
	m, _ := ModelByName("ge100pro")

	for _, tc := range []struct {
		name     string
		bank     int
		position int
		want     string
	}{
		{"bank zero", 0, 0, "has 50 banks (1..50), not bank 0"},
		{"a bank past the last", 51, 0, "has 50 banks (1..50), not bank 51"},
		{"a position past the last", 1, 3, "a bank of the Mooer GE100 Pro holds 3 presets (positions A..C), not position 4"},
	} {
		_, err := m.PresetAddress(tc.bank, tc.position)
		if err == nil {
			t.Fatalf("%s was accepted", tc.name)
		}
		if !strings.Contains(err.Error(), tc.want) {
			t.Fatalf("%s error = %q, want it to mention %q", tc.name, err, tc.want)
		}
	}
}
