package waza

// AirStepSwitch is one of the five AIRSTEP BW footswitches, labelled A–E.
type AirStepSwitch struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// AirStepBinding is what one footswitch does on a press and on a long press.
// An empty Press or LongPress means the footswitch has no action for that
// gesture.
type AirStepBinding struct {
	Switch    string `json:"switch"`
	Press     string `json:"press,omitempty"`
	LongPress string `json:"long_press,omitempty"`
}

// AirStepMode is one of the four footswitch layouts of the AIRSTEP BW.
// Indication describes how the current channel is shown in this mode.
type AirStepMode struct {
	Number     int              `json:"mode"`
	Indication string           `json:"indication,omitempty"`
	Bindings   []AirStepBinding `json:"bindings"`
}

// AirStep is the XSONIC AIRSTEP BW Edition foot controller for the Waza Air
// (and the Katana Air / Katana Go). It turns the amp's channel/patch memory
// and effect blocks into hands-free footswitches.
type AirStep struct {
	Name     string          `json:"name"`
	Display  string          `json:"display"`
	Switches []AirStepSwitch `json:"switches"`
	Modes    []AirStepMode   `json:"modes"`
}

// DefaultAirStep returns the AIRSTEP BW model with the four Waza Air modes
// transcribed from the owner's manual (v1.3).
func DefaultAirStep() AirStep {
	return AirStep{
		Name:    "airstep-bw",
		Display: "XSONIC AIRSTEP BW Edition",
		Switches: []AirStepSwitch{
			{ID: "A", Label: "A"},
			{ID: "B", Label: "B"},
			{ID: "C", Label: "C"},
			{ID: "D", Label: "D"},
			{ID: "E", Label: "E"},
		},
		Modes: []AirStepMode{
			{
				Number:     1,
				Indication: "Main LED blue = CH 1-3, green = CH 4-6.",
				Bindings: bindings(
					[3]string{"A", "Toggle BOOSTER", ""},
					[3]string{"B", "Toggle FX", ""},
					[3]string{"C", "CH 1/4", "Select CH 1-3"},
					[3]string{"D", "CH 2/5", "Select CH 4-6"},
					[3]string{"E", "CH 3/6", ""},
				),
			},
			{
				Number:     2,
				Indication: "Footswitch LED blinks green on the current channel; CH 6 = FS D and FS E blink together.",
				Bindings: bindings(
					[3]string{"A", "Toggle BOOSTER", ""},
					[3]string{"B", "Toggle FX", ""},
					[3]string{"C", "Toggle DELAY2", ""},
					[3]string{"D", "CH-", ""},
					[3]string{"E", "CH+", ""},
				),
			},
			{
				Number:     3,
				Indication: "Footswitch LED blinks green on the current channel; CH 6 = FS D and FS E blink together.",
				Bindings: bindings(
					[3]string{"A", "Toggle BOOSTER", "CH 1"},
					[3]string{"B", "Toggle MOD", "CH 2"},
					[3]string{"C", "Toggle FX", "CH 3"},
					[3]string{"D", "Toggle DELAY", "CH 4"},
					[3]string{"E", "Toggle REVERB & DELAY2", "CH 5"},
				),
			},
			{
				Number: 4,
				Bindings: bindings(
					[3]string{"A", "CH 1", ""},
					[3]string{"B", "CH 2", ""},
					[3]string{"C", "CH 3", ""},
					[3]string{"D", "CH 4", ""},
					[3]string{"E", "CH 5", "CH 6"},
				),
			},
		},
	}
}

// Mode returns one of the four footswitch layouts by number (1–4). The second
// result is false when the number is out of range.
func (a AirStep) Mode(n int) (AirStepMode, bool) {
	for _, m := range a.Modes {
		if m.Number == n {
			return m, true
		}
	}
	return AirStepMode{}, false
}

func bindings(rows ...[3]string) []AirStepBinding {
	out := make([]AirStepBinding, len(rows))
	for i, r := range rows {
		out[i] = AirStepBinding{Switch: r[0], Press: r[1], LongPress: r[2]}
	}
	return out
}
