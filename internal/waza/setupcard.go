package waza

import (
	"fmt"
	"html"
	"strings"

	"github.com/d-led/guitar-modeler-mcp/internal/cardchain"
	"github.com/d-led/guitar-modeler-mcp/internal/device"
)

// Spec is a tone to dial in on the Waza Air: the selected amp, effects and
// spatial settings. Empty fields are left off the setup card; a zero numeric
// value means "leave to BOSS TONE STUDIO" (the knob is omitted from the card
// and, on write, the template's byte is kept).
type Spec struct {
	Name             string
	Amp              string
	Booster          string
	Mod              string
	FX               string
	Delay            string
	Reverb           string
	CabResonance     string
	Ambience         string
	Position         string
	Mode             string
	Gain             int
	Volume           int
	Bass             int
	Middle           int
	Treble           int
	Presence         int
	BoosterDrive     int
	BoosterBottom    int
	BoosterTone      int
	BoosterSolo      bool
	BoosterSoloLevel int
	BoosterLevel     int
	BoosterDirectMix int
	ModParams        map[string]float64
	FXParams         map[string]float64
	DelayTime        int
	DelayFeedback    int
	DelayHighCut     int
	DelayLevel       int
	DelayDirectMix   int
	ReverbTime       float64
	ReverbPreDelay   int
	ReverbLevel      int
	ReverbDirectMix  int
	GuitarPosition   int
	AmbienceLevel    int
	NSOn             *bool
	NSThreshold      int
	NSRelease        int
}

type itemResolver struct {
	target *string
	items  []Item
	label  string
}

type stringResolver struct {
	target  *string
	options []string
}

// Resolve canonicalises the spec's selections to on-device names. Each field
// matches an exact model name first, then a substring of the "inspired by"
// description; an empty field is left empty.
func (d Device) Resolve(s Spec) (Spec, error) {
	var err error
	for _, step := range d.itemResolvers(&s) {
		if *step.target, err = device.ResolveItem(step.items, *step.target, step.label); err != nil {
			return s, err
		}
	}
	for _, step := range d.stringResolvers(&s) {
		if *step.target, err = resolveString(step.options, *step.target); err != nil {
			return s, err
		}
	}
	return s, nil
}

func (d Device) itemResolvers(s *Spec) []itemResolver {
	return []itemResolver{
		{&s.Amp, d.Amps, "amp"},
		{&s.Booster, d.Boosters, "booster"},
		{&s.Mod, d.ModFX, "mod"},
		{&s.FX, d.ModFX, "fx"},
		{&s.Delay, d.Delays, "delay"},
		{&s.Reverb, d.Reverbs, "reverb"},
	}
}

func (d Device) stringResolvers(s *Spec) []stringResolver {
	return []stringResolver{
		{&s.CabResonance, d.CabResonance},
		{&s.Ambience, d.Ambience},
		{&s.Position, d.Position},
		{&s.Mode, d.Mode},
	}
}

func resolveString(options []string, query string) (string, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		return "", nil
	}
	for _, o := range options {
		if strings.EqualFold(o, q) {
			return o, nil
		}
	}
	return "", fmt.Errorf("unknown value %q (valid: %s)", q, strings.Join(options, ", "))
}

func (d Device) inspired(name string, items []Item) string {
	for _, it := range items {
		if strings.EqualFold(it.Name, name) {
			return it.InspiredBy
		}
	}
	return ""
}

// chainHint renders the fixed signal chain with each slot's selected effect.
func chainHint(chain []string, s Spec) string {
	steps := make([]cardchain.Step, 0, len(chain))
	for i, module := range chain {
		steps = append(steps, cardchain.Step{Slot: i + 1, Module: module, Effect: effectFor(module, s)})
	}
	return cardchain.Render(steps)
}

func effectFor(module string, s Spec) string {
	switch module {
	case "BOOSTER":
		return s.Booster
	case "AMP":
		return s.Amp
	case "MOD":
		return s.Mod
	case "FX":
		return s.FX
	case "DELAY":
		return s.Delay
	case "REVERB":
		return s.Reverb
	}
	return ""
}

// SetupCardHTML renders a printable setup card for a resolved Spec.
func (d Device) SetupCardHTML(s Spec) string {
	return d.setupCardHTML(s, nil)
}

// SetupCardHTMLWithAirStep renders the setup card plus the AIRSTEP BW
// footswitch mapping for one mode.
func (d Device) SetupCardHTMLWithAirStep(s Spec, m AirStepMode) string {
	return d.setupCardHTML(s, &m)
}

func (d Device) setupCardHTML(s Spec, mode *AirStepMode) string {
	var b strings.Builder
	cardchain.Head(&b, s.Name+" — "+d.Display, `.inspired{color:#666;font-size:.85em}`)
	fmt.Fprintf(&b, "<h1>%s</h1><h2>%s — setup card</h2>", html.EscapeString(s.Name), html.EscapeString(d.Display))

	b.WriteString(chainHint(d.Chain, s))

	for i, module := range d.Chain {
		effect, inspired := d.effectAndInspired(module, s)
		writeModule(&b, module, effect, inspired, i+1, moduleKnobs(module, s))
	}

	writeSettings(&b, s)

	b.WriteString("<p class=\"inspired\">Knob values are 0&ndash;100 unless noted (ms). Knobs not listed keep the factory default (the written .tsl preserves them).</p>")

	if mode != nil {
		writeAirStepMode(&b, d, *mode)
	}

	b.WriteString("</body></html>")
	return b.String()
}

// effectAndInspired returns a chain module's selection and the real hardware
// it emulates.
func (d Device) effectAndInspired(module string, s Spec) (string, string) {
	switch module {
	case "BOOSTER":
		return s.Booster, d.inspired(s.Booster, d.Boosters)
	case "AMP":
		return s.Amp, d.inspired(s.Amp, d.Amps)
	case "MOD":
		return s.Mod, d.inspired(s.Mod, d.ModFX)
	case "FX":
		return s.FX, d.inspired(s.FX, d.ModFX)
	case "DELAY":
		return s.Delay, d.inspired(s.Delay, d.Delays)
	case "REVERB":
		return s.Reverb, d.inspired(s.Reverb, d.Reverbs)
	}
	return "", ""
}

// writeSettings renders the rig-level settings (cab resonance, ambience,
// position, mode, noise suppressor) that are not chain modules.
func writeSettings(b *strings.Builder, s Spec) {
	writeSetting(b, "CABINET RESONANCE", s.CabResonance)
	writeSetting(b, "AMBIENCE", s.Ambience)
	writeSetting(b, "POSITION", s.Position)
	writeSetting(b, "MODE", s.Mode)

	if s.NSOn != nil {
		on := "OFF"
		if *s.NSOn {
			on = "ON"
		}
		writeSetting(b, "NOISE SUPPRESSOR", on)
	}
	if s.NSThreshold != 0 {
		writeSetting(b, "NS THRESHOLD", fmt.Sprintf("%d", s.NSThreshold))
	}
	if s.NSRelease != 0 {
		writeSetting(b, "NS RELEASE", fmt.Sprintf("%d", s.NSRelease))
	}
	if s.GuitarPosition != 0 {
		writeSetting(b, "GUITAR POSITION", fmt.Sprintf("%d", s.GuitarPosition))
	}
	if s.AmbienceLevel != 0 {
		writeSetting(b, "AMBIENCE LEVEL", fmt.Sprintf("%d", s.AmbienceLevel))
	}
}

// trimFloat renders a float knob value without a trailing ".0".
func trimFloat(v float64) string {
	if v == float64(int(v)) {
		return fmt.Sprintf("%d", int(v))
	}
	return fmt.Sprintf("%g", v)
}

func writeAirStepMode(b *strings.Builder, d Device, m AirStepMode) {
	if m.Number == 0 || len(m.Bindings) == 0 {
		return
	}
	fmt.Fprintf(b, "<h2>AIRSTEP BW — Mode %d</h2>", m.Number)
	fmt.Fprintf(b, "<p class=\"inspired\">%s (hold A, B, C or B+C while powering on to select a mode)</p>", html.EscapeString(m.Indication))
	b.WriteString("<table><tr><th>Switch</th><th>Press</th><th>Long press</th></tr>")
	for _, bi := range m.Bindings {
		press, longPress := bi.Press, bi.LongPress
		if press == "" {
			press = "—"
		}
		if longPress == "" {
			longPress = "—"
		}
		fmt.Fprintf(b, "<tr><td class=\"module\">%s</td><td>%s</td><td>%s</td></tr>",
			html.EscapeString(bi.Switch), html.EscapeString(press), html.EscapeString(longPress))
	}
	b.WriteString("</table>")
}

// cardKnob is one editable value shown on the card, grouped under its module.
type cardKnob struct{ name, value string }

// intKnob returns a knob row for a non-zero integer (zero means "left to BOSS
// TONE STUDIO", so it is omitted).
func intKnob(name string, v int) cardKnob {
	if v == 0 {
		return cardKnob{}
	}
	return cardKnob{name, fmt.Sprintf("%d", v)}
}

// dropEmpty removes zero/empty knob rows.
func dropEmpty(ks []cardKnob) []cardKnob {
	out := make([]cardKnob, 0, len(ks))
	for _, k := range ks {
		if k.name != "" {
			out = append(out, k)
		}
	}
	return out
}

// paramKnobs renders the MOD/FX effect sub-parameters in a fixed, readable
// order, skipping unset (zero) values.
func paramKnobs(params map[string]float64) []cardKnob {
	var ks []cardKnob
	for _, name := range []string{"rate", "depth", "effect_level", "direct_mix", "level", "manual", "resonance", "sustain", "attack", "threshold", "release", "feedback"} {
		if v, ok := params[name]; ok && v != 0 {
			ks = append(ks, cardKnob{strings.ToUpper(name), trimFloat(v)})
		}
	}
	return ks
}

// moduleKnobs returns the dialled knob values for one chain module, so the
// card groups each block's settings under the block itself.
func moduleKnobs(module string, s Spec) []cardKnob {
	switch module {
	case "BOOSTER":
		return boosterKnobs(s)
	case "AMP":
		return ampKnobs(s)
	case "MOD":
		return paramKnobs(s.ModParams)
	case "FX":
		return paramKnobs(s.FXParams)
	case "DELAY":
		return delayKnobs(s)
	case "REVERB":
		return reverbKnobs(s)
	}
	return nil
}

func boosterKnobs(s Spec) []cardKnob {
	var ks []cardKnob
	ks = append(ks,
		intKnob("DRIVE", s.BoosterDrive),
		intKnob("BOTTOM", s.BoosterBottom),
		intKnob("TONE", s.BoosterTone),
		intKnob("LEVEL", s.BoosterLevel),
		intKnob("DIRECT MIX", s.BoosterDirectMix),
	)
	if s.BoosterSolo {
		ks = append(ks, cardKnob{"SOLO", fmt.Sprintf("ON (%d)", s.BoosterSoloLevel)})
	}
	return dropEmpty(ks)
}

func ampKnobs(s Spec) []cardKnob {
	return dropEmpty([]cardKnob{
		intKnob("GAIN", s.Gain),
		intKnob("VOLUME", s.Volume),
		intKnob("BASS", s.Bass),
		intKnob("MIDDLE", s.Middle),
		intKnob("TREBLE", s.Treble),
		intKnob("PRESENCE", s.Presence),
	})
}

func delayKnobs(s Spec) []cardKnob {
	var ks []cardKnob
	if s.DelayTime > 0 {
		ks = append(ks, cardKnob{"TIME", fmt.Sprintf("%d ms", s.DelayTime)})
	}
	ks = append(ks,
		intKnob("FEEDBACK", s.DelayFeedback),
		intKnob("HIGH CUT", s.DelayHighCut),
		intKnob("LEVEL", s.DelayLevel),
		intKnob("DIRECT MIX", s.DelayDirectMix),
	)
	return dropEmpty(ks)
}

func reverbKnobs(s Spec) []cardKnob {
	var ks []cardKnob
	if s.ReverbTime > 0 {
		ks = append(ks, cardKnob{"TIME", fmt.Sprintf("%.1f s", s.ReverbTime)})
	}
	if s.ReverbPreDelay > 0 {
		ks = append(ks, cardKnob{"PRE DELAY", fmt.Sprintf("%d ms", s.ReverbPreDelay)})
	}
	ks = append(ks,
		intKnob("LEVEL", s.ReverbLevel),
		intKnob("DIRECT MIX", s.ReverbDirectMix),
	)
	return dropEmpty(ks)
}

func writeModule(b *strings.Builder, module, effect, inspired string, slot int, knobs []cardKnob) {
	if effect == "" {
		effect = "OFF"
	}
	fmt.Fprintf(b, "<table><tr><td class=\"module\"><span class=\"slotbadge\">%d</span>%s</td><td class=\"effect\">%s</td></tr>", slot, html.EscapeString(module), html.EscapeString(effect))
	if inspired != "" {
		fmt.Fprintf(b, "<tr><td></td><td><span class=\"inspired\">based on %s</span></td></tr>", html.EscapeString(inspired))
	}
	for _, k := range knobs {
		fmt.Fprintf(b, "<tr><td>%s</td><td>%s</td></tr>", html.EscapeString(k.name), html.EscapeString(k.value))
	}
	b.WriteString("</table>")
}

func writeSetting(b *strings.Builder, name, value string) {
	if value == "" {
		return
	}
	fmt.Fprintf(b, "<table><tr><td class=\"module\">%s</td><td class=\"effect\">%s</td></tr></table>", html.EscapeString(name), html.EscapeString(value))
}
