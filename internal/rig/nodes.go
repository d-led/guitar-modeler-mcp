package rig

import "strconv"

// fixedNodes are the structural Patch sections that every rig contains. They
// are reproduced from a factory rig so the produced file matches what the
// device expects.

func rigNode(name string, tempo float64) *Node {
	n := newNode("PresetName", "Tempo", "TempoFromMaster", "ExtAmp", "RigCurPedalAB")
	n.set("PresetName", label(name))
	n.set("Tempo", num(tempo))
	n.set("TempoFromMaster", boolean(false))
	n.set("ExtAmp", str("None"))
	n.set("RigCurPedalAB", boolean(false))
	return n
}

func inputNode(inputGain float64) *Node {
	n := newNode("PresetName", "InputGain", "GateThresh", "GateRelease", "FilterThreshold")
	n.set("PresetName", label(""))
	n.set("InputGain", num(inputGain))
	n.set("GateThresh", num(-60))
	n.set("GateRelease", num(11))
	n.set("FilterThreshold", num(-120))
	return n
}

func outputNode() *Node {
	n := newNode("PresetName", "RigVolume", "RigWidth")
	n.set("PresetName", label(""))
	n.set("RigVolume", num(0))
	n.set("RigWidth", num(100))
	return n
}

func mixNode() *Node {
	n := newNode("Para1Level", "Para2Level", "Para1Pan", "Para2Pan", "ParaDelay", "PresetName")
	n.set("Para1Level", num(-6))
	n.set("Para2Level", num(-6))
	n.set("Para1Pan", num(0))
	n.set("Para2Pan", num(0))
	n.set("ParaDelay", num(0))
	n.set("PresetName", label(""))
	return n
}

func chainNode(moduleNames []string) *Node {
	order := []string{"Routing", "Tails"}
	for i := 1; i <= 11; i++ {
		order = append(order, "ModuleType"+strconv.Itoa(i))
	}
	order = append(order, "Para1Level", "Para2Level", "Para1Pan", "Para2Pan", "ParaDelay")

	n := newNode(order...)
	n.set("Routing", str("S"))
	n.set("Tails", boolean(true))
	for i := 1; i <= 11; i++ {
		name := "Empty Slot"
		if i <= len(moduleNames) {
			name = moduleNames[i-1]
		}
		n.set("ModuleType"+strconv.Itoa(i), str(name))
	}
	n.set("Para1Level", num(-6))
	n.set("Para2Level", num(-6))
	n.set("Para1Pan", num(0))
	n.set("Para2Pan", num(0))
	n.set("ParaDelay", num(0))
	return n
}

// ampNode builds the Amp module. The amp model doubles as the emulated
// amplifier selection ("Type") and supports the device's double-amp states.
func ampNode(model string, params map[string]any) *Node {
	n := newNode(
		"PresetName", "PresetName2", "TremSync", "Doubling", "DoubleStates", "On", "Colour", "Type",
		"Bright", "Treble", "Mid", "MidFreq", "Bass", "Presence", "GainA", "GainB", "TremSpeed", "TremDepth", "Master", "Tremolo",
		"Type2", "Bright2", "Treble2", "Mid2", "MidFreq2", "Bass2", "Presence2", "GainA2", "GainB2", "TremSpeed2", "TremDepth2", "Master2",
	)
	n.set("PresetName", label(""))
	n.set("PresetName2", label(""))
	n.set("TremSync", boolean(false))
	n.set("Doubling", boolean(false))
	n.set("DoubleStates", dblState(false))
	n.set("On", boolean(true))
	n.set("Colour", str("Green"))
	n.set("Type", str(model))
	n.set("Bright", boolean(false))
	n.set("Treble", num(50))
	n.set("Mid", num(50))
	n.set("MidFreq", num(1610))
	n.set("Bass", num(50))
	n.set("Presence", num(50))
	n.set("GainA", num(50))
	n.set("GainB", num(50))
	n.set("TremSpeed", num(2))
	n.set("TremDepth", num(60))
	n.set("Master", num(50))
	n.set("Tremolo", boolean(false))
	n.set("Type2", str(""))
	n.set("Bright2", boolean(true))
	n.set("Treble2", num(50))
	n.set("Mid2", num(50))
	n.set("MidFreq2", num(1610))
	n.set("Bass2", num(50))
	n.set("Presence2", num(50))
	n.set("GainA2", num(50))
	n.set("GainB2", num(50))
	n.set("TremSpeed2", num(0.41))
	n.set("TremDepth2", num(0))
	n.set("Master2", num(50))
	applyParams(n.Children, params)
	return n
}

// cabNode builds the Cab module with the selected cabinet and microphone.
func cabNode(cabModel, micModel string, params map[string]any) *Node {
	n := newNode(
		"PresetName", "PresetName2", "CabType", "CabType2", "Doubling", "DoubleStates", "On", "Colour",
		"MicType", "MicType2", "OnAxis", "OnAxis2", "Breakup", "Breakup2", "OutGain", "OutGain2", "AmpCompGain",
	)
	n.set("PresetName", label(""))
	n.set("PresetName2", label(""))
	n.set("CabType", str(cabModel))
	n.set("CabType2", str(""))
	n.set("Doubling", boolean(false))
	n.set("DoubleStates", dblState(false))
	n.set("On", boolean(true))
	n.set("Colour", str("Green"))
	n.set("MicType", str(micModel))
	n.set("MicType2", str("Dyn 7"))
	n.set("OnAxis", boolean(true))
	n.set("OnAxis2", boolean(false))
	n.set("Breakup", num(0))
	n.set("Breakup2", num(0))
	n.set("OutGain", num(0))
	n.set("OutGain2", num(0))
	n.set("AmpCompGain", num(0))
	applyParams(n.Children, params)
	return n
}
