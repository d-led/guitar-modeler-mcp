package catalog

// mics is the complete list of microphone models available for cabinet
// emulation on the HeadRush Gigboard.
var mics = []Mic{
	{Model: "Dyn 57", Kind: "dynamic", RealModel: "Shure SM57", Description: "The classic close-mic dynamic workhorse."},
	{Model: "Dyn 7", Kind: "dynamic", RealModel: "Shure SM7B", Description: "Smooth, fat dynamic broadcast microphone."},
	{Model: "Dyn 409", Kind: "dynamic", RealModel: "Sennheiser MD409", Description: "Bright, tight dynamic with a fast transient."},
	{Model: "Dyn 421", Kind: "dynamic", RealModel: "Sennheiser MD421", Description: "Full-bodied dynamic with strong low end."},
	{Model: "Dyn 20", Kind: "dynamic", RealModel: "Electro-Voice RE20", Description: "Large-diaphragm dynamic with a neutral, even response."},
	{Model: "Cond 414", Kind: "condenser", RealModel: "AKG C414", Description: "Detailed large-diaphragm condenser."},
	{Model: "Cond 67", Kind: "condenser", RealModel: "Neumann U67", Description: "Vintage tube condenser with a warm top end."},
	{Model: "Cond 87", Kind: "condenser", RealModel: "Neumann U87", Description: "Studio-standard large-diaphragm condenser."},
	{Model: "Ribbon 121", Kind: "ribbon", RealModel: "Royer R-121", Description: "Smooth ribbon microphone, dark and natural."},
}
