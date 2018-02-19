package main

// Parameters is a set of parameters that control the visualization
type Parameters struct {
	GlobalBrightness float64
	Brightness       float64
	Direction        int
	Gain             float64
	DifferentialGain float64
	Offset           float64
	Period           int
	Sync             float64
}

// Config is passed to initialize the module
type Config struct {
	Buckets    int
	Columns    int
	SampleRate float64
	Parameters Parameters
}
