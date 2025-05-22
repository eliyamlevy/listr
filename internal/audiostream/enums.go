package audiostream

// FrequencyBand represents the different frequency bands used in Shazam signatures
type FrequencyBand int

const (
	// Define frequency bands (these values should match the Python implementation)
	LowBand      FrequencyBand = 0
	MidBand      FrequencyBand = 1
	HighBand     FrequencyBand = 2
	VeryHighBand FrequencyBand = 3
)

// SampleRate represents the supported sample rates
type SampleRate int

const (
	SampleRate8000  SampleRate = 8000
	SampleRate16000 SampleRate = 16000
	SampleRate32000 SampleRate = 32000
	SampleRate44100 SampleRate = 44100
	SampleRate48000 SampleRate = 48000
)
