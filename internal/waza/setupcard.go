package waza

import (
	"fmt"
	"html"
	"strings"

	"github.com/d-led/guitar-modeler-mcp/internal/cardchain"
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

// Resolve canonicalises the spec's selections to on-device names. Each field
// matches an exact model name first, then a substring of the "inspired by"
// description; an empty field is left empty.
func (d Device) Resolve(s Spec) (Spec, error) {
	var err error
	if s.Amp, err = resolve(d.Amps, s.Amp, "amp"); err != nil {
		return s, err
	}
	if s.Booster, err = resolve(d.Boosters, s.Booster, "booster"); err != nil {
		return s, err
	}
	if s.Mod, err = resolve(d.ModFX, s.Mod, "mod"); err != nil {
		return s, err
	}
	if s.FX, err = resolve(d.ModFX, s.FX, "fx"); err != nil {
		return s, err
	}
	if s.Delay, err = resolve(d.Delays, s.Delay, "delay"); err != nil {
		return s, err
	}
	if s.Reverb, err = resolve(d.Reverbs, s.Reverb, "reverb"); err != nil {
		return s, err
	}
	if s.CabResonance, err = resolveString(d.CabResonance, s.CabResonance); err != nil {
		return s, err
	}
	if s.Ambience, err = resolveString(d.Ambience, s.Ambience); err != nil {
		return s, err
	}
	if s.Position, err = resolveString(d.Position, s.Position); err != nil {
		return s, err
	}
	if s.Mode, err = resolveString(d.Mode, s.Mode); err != nil {
		return s, err
	}
	return s, nil
}

func resolve(items []Item, query, label string) (string, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		return "", nil
	}
	for _, it := range items {
		if strings.EqualFold(it.Name, q) {
			return it.Name, nil
		}
	}
	for _, it := range items {
		if it.InspiredBy != "" && strings.Contains(strings.ToLower(it.InspiredBy), strings.ToLower(q)) {
			return it.Name, nil
		}
	}
	return "", fmt.Errorf("no %s matches %q", label, q)
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
	b.WriteString("<!doctype html><html><head><meta charset=\"utf-8\">")
	fmt.Fprintf(&b, "<title>%s — %s</title>", html.EscapeString(s.Name), html.EscapeString(d.Display))
	b.WriteString(`<style>
body{font-family:system-ui,-apple-system,sans-serif;max-width:720px;margin:2rem auto;padding:0 1rem;color:#1a1a1a}
h1{margin-bottom:.25rem}h2{font-size:1rem;color:#555;margin-top:1.25rem}
table{width:100%;border-collapse:collapse;margin-bottom:1rem}
td,th{border-bottom:1px solid #e2e2e2;padding:.45rem .5rem;text-align:left;vertical-align:top}
.module{font-weight:600;white-space:nowrap}
.effect{font-weight:600}.inspired{color:#666;font-size:.85em}
` + cardchain.CSS + `
</style></head><body>`)
	fmt.Fprintf(&b, "<h1>%s</h1><h2>%s — setup card</h2>", html.EscapeString(s.Name), html.EscapeString(d.Display))

	b.WriteString(chainHint(d.Chain, s))

	for i, module := range d.Chain {
		var effect, inspired string
		switch module {
		case "BOOSTER":
			effect, inspired = s.Booster, d.inspired(s.Booster, d.Boosters)
		case "AMP":
			effect, inspired = s.Amp, d.inspired(s.Amp, d.Amps)
		case "MOD":
			effect, inspired = s.Mod, d.inspired(s.Mod, d.ModFX)
		case "FX":
			effect, inspired = s.FX, d.inspired(s.FX, d.ModFX)
		case "DELAY":
			effect, inspired = s.Delay, d.inspired(s.Delay, d.Delays)
		case "REVERB":
			effect, inspired = s.Reverb, d.inspired(s.Reverb, d.Reverbs)
		}
		writeModule(&b, module, effect, inspired, i+1)
	}

	writeSetting(&b, "CABINET RESONANCE", s.CabResonance)
	writeSetting(&b, "AMBIENCE", s.Ambience)
	writeSetting(&b, "POSITION", s.Position)
	writeSetting(&b, "MODE", s.Mode)

	writeParams(&b, s)

	b.WriteString("<p class=\"inspired\">Knob values are 0&ndash;100 unless noted (ms). Knobs not listed keep the factory default (the written .tsl preserves them).</p>")

	if mode != nil {
		writeAirStepMode(&b, d, *mode)
	}

	b.WriteString("</body></html>")
	return b.String()
}

// writeParams renders the specified knob values on the card. A zero value is
// omitted (it means "not specified").
func writeParams(b *strings.Builder, s Spec) {
	var rows []struct{ name, value string }
	add := func(name string, v int) {
		if v != 0 {
			rows = append(rows, struct{ name, value string }{name, fmt.Sprintf("%d", v)})
		}
	}
	add("AMP GAIN", s.Gain)
	add("AMP VOLUME", s.Volume)
	add("AMP BASS", s.Bass)
	add("AMP MIDDLE", s.Middle)
	add("AMP TREBLE", s.Treble)
	add("AMP PRESENCE", s.Presence)
	add("BOOSTER DRIVE", s.BoosterDrive)
	add("BOOSTER BOTTOM", s.BoosterBottom)
	add("BOOSTER TONE", s.BoosterTone)
	add("BOOSTER LEVEL", s.BoosterLevel)
	add("BOOSTER DIRECT MIX", s.BoosterDirectMix)
	if s.BoosterSolo {
		rows = append(rows, struct{ name, value string }{"BOOSTER SOLO", fmt.Sprintf("ON (%d)", s.BoosterSoloLevel)})
	}
	for _, name := range []string{"rate", "depth", "effect_level", "direct_mix", "level", "manual", "resonance", "sustain", "attack", "threshold", "release", "feedback"} {
		if v, ok := s.ModParams[name]; ok && v != 0 {
			rows = append(rows, struct{ name, value string }{strings.ToUpper(name), trimFloat(v)})
		}
	}
	if s.DelayTime > 0 {
		rows = append(rows, struct{ name, value string }{"DELAY TIME", fmt.Sprintf("%d ms", s.DelayTime)})
	}
	add("DELAY FEEDBACK", s.DelayFeedback)
	add("DELAY HIGH CUT", s.DelayHighCut)
	add("DELAY LEVEL", s.DelayLevel)
	add("DELAY DIRECT MIX", s.DelayDirectMix)
	if s.ReverbTime > 0 {
		rows = append(rows, struct{ name, value string }{"REVERB TIME", fmt.Sprintf("%.1f s", s.ReverbTime)})
	}
	if s.ReverbPreDelay > 0 {
		rows = append(rows, struct{ name, value string }{"REVERB PRE DELAY", fmt.Sprintf("%d ms", s.ReverbPreDelay)})
	}
	add("REVERB LEVEL", s.ReverbLevel)
	add("REVERB DIRECT MIX", s.ReverbDirectMix)
	add("GUITAR POSITION", s.GuitarPosition)
	add("AMBIENCE LEVEL", s.AmbienceLevel)
	if s.NSOn != nil {
		on := "OFF"
		if *s.NSOn {
			on = "ON"
		}
		rows = append(rows, struct{ name, value string }{"NOISE SUPPRESSOR", on})
	}
	add("NS THRESHOLD", s.NSThreshold)
	add("NS RELEASE", s.NSRelease)

	for _, r := range rows {
		writeSetting(b, r.name, r.value)
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

func writeModule(b *strings.Builder, module, effect, inspired string, slot int) {
	if effect == "" {
		effect = "OFF"
	}
	fmt.Fprintf(b, "<table><tr><td class=\"module\"><span class=\"slotbadge\">%d</span>%s</td><td class=\"effect\">%s</td></tr>", slot, html.EscapeString(module), html.EscapeString(effect))
	if inspired != "" {
		fmt.Fprintf(b, "<tr><td></td><td><span class=\"inspired\">based on %s</span></td></tr>", html.EscapeString(inspired))
	}
	b.WriteString("</table>")
}

func writeSetting(b *strings.Builder, name, value string) {
	if value == "" {
		return
	}
	fmt.Fprintf(b, "<table><tr><td class=\"module\">%s</td><td class=\"effect\">%s</td></tr></table>", html.EscapeString(name), html.EscapeString(value))
}
