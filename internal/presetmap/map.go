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
		switch {
		case strings.HasPrefix(slot, "Amp"):
			if model, ok := m.Params["Type"].(string); ok {
				if mooerModel, found := t.MapAmp(DeviceGigboard, model, DeviceMooer); found {
					if index, found := mooer.AmpIndex(mooerModel); found {
						p.Amp = mooer.Amp{Enabled: m.On, Type: index}
					}
				}
			}
		case strings.HasPrefix(slot, "Cab"):
			if model, ok := m.Params["CabType"].(string); ok {
				if mooerModel, found := t.MapCab(DeviceGigboard, model, DeviceMooer); found {
					if index, found := mooer.CabIndex(mooerModel); found {
						p.Cab = mooer.Cab{Enabled: m.On, Type: index}
					}
				}
			}
		case slot == "Gate" || slot == "Noise Filter":
			p.NoiseGate.Enabled = m.On
		default:
			if module, name, ok := t.MapFXGigboardToMooer(slot); ok {
				if index, found := mooer.EffectIndex(module, name); found {
					setMooerModule(&p, module, index, m.On)
				}
			}
		}
	}
	return p, nil
}

// MooerToGigboard maps a Mooer preset to a buildable Gigboard rig spec. As with
// GigboardToMooer, the mapping is structural (which modules are present and
// on); parameter values are not translated.
func (t *Table) MooerToGigboard(p mooer.Preset) (rig.Spec, error) {
	ampModel, ok := t.MapAmp(DeviceMooer, mooer.EffectName("amp", p.Amp.Type), DeviceGigboard)
	if !ok {
		return rig.Spec{}, &UnmappedError{Kind: "amp", Model: mooer.EffectName("amp", p.Amp.Type)}
	}
	cabModel, ok := t.MapCab(DeviceMooer, mooer.EffectName("cab", p.Cab.Type), DeviceGigboard)
	if !ok {
		return rig.Spec{}, &UnmappedError{Kind: "cab", Model: mooer.EffectName("cab", p.Cab.Type)}
	}

	blocks := make([]rig.Block, 0, 9)
	blocks = append(blocks, mappedFX(t, "fx", mooer.EffectName("fx", p.FX.Type), p.FX.Enabled)...)
	blocks = append(blocks, mappedFX(t, "od", mooer.EffectName("od", p.Drive.Type), p.Drive.Enabled)...)
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

	blocks = append(blocks, mappedFX(t, "mod", mooer.EffectName("mod", p.Mod.Type), p.Mod.Enabled)...)
	blocks = append(blocks, mappedFX(t, "delay", mooer.EffectName("delay", p.Delay.Type), p.Delay.Enabled)...)
	blocks = append(blocks, mappedFX(t, "reverb", mooer.EffectName("reverb", p.Reverb.Type), p.Reverb.Enabled)...)

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

func setMooerModule(p *mooer.Preset, module string, index uint8, enabled bool) {
	switch module {
	case "fx":
		p.FX = mooer.FX{Enabled: enabled, Type: index}
	case "od":
		p.Drive = mooer.Drive{Enabled: enabled, Type: index}
	case "mod":
		p.Mod = mooer.Mod{Enabled: enabled, Type: index}
	case "delay":
		p.Delay = mooer.Delay{Enabled: enabled, Type: index}
	case "reverb":
		p.Reverb = mooer.Reverb{Enabled: enabled, Type: index}
	}
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
