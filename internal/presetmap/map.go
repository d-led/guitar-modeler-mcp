package presetmap

import (
	"strings"

	"github.com/d-led/guitar-modeler-mcp/internal/mooer"
	"github.com/d-led/guitar-modeler-mcp/internal/rig"
)

// GigboardToMooer maps a decoded Gigboard rig to a Mooer preset. The mapping
// is structural: which amp, cab and effects are in the chain and whether each
// is on. Parameter values are not translated — the two devices scale their
// parameters differently and to different ranges — so the Mooer modules carry
// their default parameter values.
func (t *Table) GigboardToMooer(file *rig.RigFile) (mooer.Preset, error) {
	summary, err := rig.Describe(file)
	if err != nil {
		return mooer.Preset{}, err
	}

	p := mooer.New()
	p.Name = summary.Name

	modules := make(map[string]rig.SummaryModule, len(summary.Modules))
	for _, m := range summary.Modules {
		modules[m.Name] = m
	}

	for _, slot := range summary.Slots {
		m, ok := modules[slot]
		if !ok {
			continue
		}
		t.applySlot(&p, slot, m)
	}
	return p, nil
}

// applySlot maps one chain slot onto the corresponding Mooer module.
func (t *Table) applySlot(p *mooer.Preset, slot string, m rig.SummaryModule) {
	switch {
	case strings.HasPrefix(slot, "Amp"):
		t.applyAmp(p, m)
	case strings.HasPrefix(slot, "Cab"):
		t.applyCab(p, m)
	case slot == "Gate" || slot == "Noise Filter":
		mooer.SetModule(p, "ns", 0, m.On)
	default:
		t.applyFX(p, slot, m)
	}
}

func (t *Table) applyAmp(p *mooer.Preset, m rig.SummaryModule) {
	model, ok := m.Params["Type"].(string)
	if !ok {
		return
	}
	mooerModel, found := t.MapAmp(DeviceGigboard, model, DeviceMooer)
	if !found {
		return
	}
	index, found := t.mooer.AmpIndex(mooerModel)
	if !found {
		return
	}
	mooer.SetModule(p, "amp", index, m.On)
}

func (t *Table) applyCab(p *mooer.Preset, m rig.SummaryModule) {
	model, ok := m.Params["CabType"].(string)
	if !ok {
		return
	}
	mooerModel, found := t.MapCab(DeviceGigboard, model, DeviceMooer)
	if !found {
		return
	}
	index, found := t.mooer.CabIndex(mooerModel)
	if !found {
		return
	}
	mooer.SetModule(p, "cab", index, m.On)
}

func (t *Table) applyFX(p *mooer.Preset, slot string, m rig.SummaryModule) {
	module, name, ok := t.MapFXGigboardToMooer(slot)
	if !ok {
		return
	}
	index, found := t.mooer.EffectIndex(module, name)
	if !found {
		return
	}
	mooer.SetModule(p, module, index, m.On)
}

// MooerToGigboard maps a Mooer preset to a buildable Gigboard rig spec. As with
// GigboardToMooer, the mapping is structural (which modules are present and
// on); parameter values are not translated.
func (t *Table) MooerToGigboard(p mooer.Preset) (rig.Spec, error) {
	ampModel, ok := t.MapAmp(DeviceMooer, t.mooer.EffectName("amp", p.Amp.Type), DeviceGigboard)
	if !ok {
		return rig.Spec{}, &UnmappedError{Kind: "amp", Model: t.mooer.EffectName("amp", p.Amp.Type)}
	}
	cabModel, ok := t.MapCab(DeviceMooer, t.mooer.EffectName("cab", p.Cab.Type), DeviceGigboard)
	if !ok {
		return rig.Spec{}, &UnmappedError{Kind: "cab", Model: t.mooer.EffectName("cab", p.Cab.Type)}
	}

	blocks := make([]rig.Block, 0, 9)
	blocks = append(blocks, mappedFX(t, "fx", t.mooer.EffectName("fx", p.FX.Type), p.FX.Enabled)...)
	blocks = append(blocks, mappedFX(t, "od", t.mooer.EffectName("od", p.Drive.Type), p.Drive.Enabled)...)
	if p.NoiseGate.Enabled {
		blocks = append(blocks, rig.Block{Type: "Gate", Enabled: true})
	}
	if p.EQ.Enabled {
		blocks = append(blocks, rig.Block{Type: "Graphic EQ", Enabled: true})
	}

	blocks = append(blocks,
		rig.Block{Type: "Amp", Enabled: true, Params: map[string]any{"Type": ampModel, "On": p.Amp.Enabled}},
		rig.Block{Type: "Cab", Enabled: true, Params: map[string]any{"CabType": cabModel, "MicType": "Dyn 57", "On": p.Cab.Enabled}},
	)

	blocks = append(blocks, mappedFX(t, "mod", t.mooer.EffectName("mod", p.Mod.Type), p.Mod.Enabled)...)
	blocks = append(blocks, mappedFX(t, "delay", t.mooer.EffectName("delay", p.Delay.Type), p.Delay.Enabled)...)
	blocks = append(blocks, mappedFX(t, "reverb", t.mooer.EffectName("reverb", p.Reverb.Type), p.Reverb.Enabled)...)

	return rig.Spec{Name: p.Name, Blocks: blocks}, nil
}

// mappedFX translates one enabled Mooer module to a Gigboard effect block.
func mappedFX(t *Table, module, name string, enabled bool) []rig.Block {
	if !enabled {
		return nil
	}
	gig, ok := t.MapFXMooerToGigboard(module, name)
	if !ok {
		return nil
	}
	return []rig.Block{{Type: gig, Enabled: true}}
}

// UnmappedError reports that a Mooer model has no Gigboard counterpart, so the
// mapping cannot produce a valid rig.
type UnmappedError struct {
	Kind  string
	Model string
}

func (e *UnmappedError) Error() string {
	return "no Gigboard " + e.Kind + " emulates Mooer " + e.Kind + " model \"" + e.Model + "\""
}
