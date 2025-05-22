package audiostream

import (
	"encoding/base64"
	"testing"
)

func TestFrequencyPeakCalculations(t *testing.T) {
	tests := []struct {
		name              string
		peak              FrequencyPeak
		expectedFrequency float64
		expectedAmplitude float64
		expectedSeconds   float64
	}{
		{
			name: "Standard 16kHz sample rate",
			peak: FrequencyPeak{
				FFTPassNumber:             100,
				PeakMagnitude:             7000,
				CorrectedPeakFrequencyBin: 512,
				SampleRateHz:              16000,
			},
			expectedFrequency: 62.5, // Actual calculation: 512 * (16000/2/1024/64) = 512 * 0.1220703125
			expectedAmplitude: 0.0,  // This will be calculated
			expectedSeconds:   0.8,  // (100 * 128) / 16000
		},
		{
			name: "44.1kHz sample rate",
			peak: FrequencyPeak{
				FFTPassNumber:             200,
				PeakMagnitude:             6500,
				CorrectedPeakFrequencyBin: 256,
				SampleRateHz:              44100,
			},
			expectedFrequency: 86.1328125,         // Actual calculation: 256 * (44100/2/1024/64) = 256 * 0.33642578125
			expectedAmplitude: 0.0,                // This will be calculated
			expectedSeconds:   0.5804988662131519, // (200 * 128) / 44100
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test GetFrequencyHz
			freq := tt.peak.GetFrequencyHz()
			if !floatEquals(freq, tt.expectedFrequency) {
				t.Errorf("GetFrequencyHz() = %v, want %v", freq, tt.expectedFrequency)
			}

			// Test GetAmplitudePCM
			amp := tt.peak.GetAmplitudePCM()
			if amp <= 0 {
				t.Errorf("GetAmplitudePCM() returned non-positive value: %v", amp)
			}

			// Test GetSeconds
			secs := tt.peak.GetSeconds()
			if !floatEquals(secs, tt.expectedSeconds) {
				t.Errorf("GetSeconds() = %v, want %v", secs, tt.expectedSeconds)
			}
		})
	}
}

func TestDecodeEncodeRoundTrip(t *testing.T) {
	// Create a sample message
	msg := &DecodedMessage{
		SampleRateHz:  16000,
		NumberSamples: 1000,
		FrequencyBandToSoundPeaks: map[FrequencyBand][]FrequencyPeak{
			LowBand: {
				{
					FFTPassNumber:             100,
					PeakMagnitude:             7000,
					CorrectedPeakFrequencyBin: 512,
					SampleRateHz:              16000,
				},
			},
			MidBand: {
				{
					FFTPassNumber:             200,
					PeakMagnitude:             6500,
					CorrectedPeakFrequencyBin: 256,
					SampleRateHz:              16000,
				},
			},
		},
	}

	// Test binary encoding/decoding
	t.Run("Binary Round Trip", func(t *testing.T) {
		// Encode to binary
		binary, err := msg.EncodeToBinary()
		if err != nil {
			t.Fatalf("EncodeToBinary() error = %v", err)
		}

		// Decode from binary
		decoded, err := DecodeFromBinary(binary)
		if err != nil {
			t.Fatalf("DecodeFromBinary() error = %v", err)
		}

		// Compare original and decoded messages
		if decoded.SampleRateHz != msg.SampleRateHz {
			t.Errorf("SampleRateHz = %v, want %v", decoded.SampleRateHz, msg.SampleRateHz)
		}
		if decoded.NumberSamples != msg.NumberSamples {
			t.Errorf("NumberSamples = %v, want %v", decoded.NumberSamples, msg.NumberSamples)
		}

		// Compare frequency peaks
		for band, peaks := range msg.FrequencyBandToSoundPeaks {
			decodedPeaks, exists := decoded.FrequencyBandToSoundPeaks[band]
			if !exists {
				t.Errorf("Frequency band %v not found in decoded message", band)
				continue
			}
			if len(decodedPeaks) != len(peaks) {
				t.Errorf("Number of peaks for band %v = %v, want %v", band, len(decodedPeaks), len(peaks))
				continue
			}
			for i, peak := range peaks {
				if decodedPeaks[i].FFTPassNumber != peak.FFTPassNumber {
					t.Errorf("FFTPassNumber = %v, want %v", decodedPeaks[i].FFTPassNumber, peak.FFTPassNumber)
				}
				if decodedPeaks[i].PeakMagnitude != peak.PeakMagnitude {
					t.Errorf("PeakMagnitude = %v, want %v", decodedPeaks[i].PeakMagnitude, peak.PeakMagnitude)
				}
				if decodedPeaks[i].CorrectedPeakFrequencyBin != peak.CorrectedPeakFrequencyBin {
					t.Errorf("CorrectedPeakFrequencyBin = %v, want %v", decodedPeaks[i].CorrectedPeakFrequencyBin, peak.CorrectedPeakFrequencyBin)
				}
			}
		}
	})

	// Test URI encoding/decoding
	t.Run("URI Round Trip", func(t *testing.T) {
		// Encode to URI
		uri, err := msg.EncodeToURI()
		if err != nil {
			t.Fatalf("EncodeToURI() error = %v", err)
		}

		// Verify URI prefix
		if uri[:len(DataURIPrefix)] != DataURIPrefix {
			t.Errorf("URI prefix = %v, want %v", uri[:len(DataURIPrefix)], DataURIPrefix)
		}

		// Decode base64 part
		binary, err := base64.StdEncoding.DecodeString(uri[len(DataURIPrefix):])
		if err != nil {
			t.Fatalf("base64 decode error = %v", err)
		}

		// Decode binary
		decoded, err := DecodeFromBinary(binary)
		if err != nil {
			t.Fatalf("DecodeFromBinary() error = %v", err)
		}

		// Compare original and decoded messages
		if decoded.SampleRateHz != msg.SampleRateHz {
			t.Errorf("SampleRateHz = %v, want %v", decoded.SampleRateHz, msg.SampleRateHz)
		}
	})
}

func TestInvalidBinaryData(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		wantErr bool
	}{
		{
			name:    "Empty data",
			data:    []byte{},
			wantErr: true,
		},
		{
			name:    "Too short data",
			data:    make([]byte, 10),
			wantErr: true,
		},
		{
			name:    "Invalid magic1",
			data:    make([]byte, 100),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DecodeFromBinary(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("DecodeFromBinary() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Helper function to compare float64 values with a small epsilon
func floatEquals(a, b float64) bool {
	epsilon := 0.0001
	return (a-b) < epsilon && (b-a) < epsilon
}
