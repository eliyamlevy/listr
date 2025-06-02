package shazam

import (
	"bytes"
	"encoding/json"
	"fmt"
	"listr/internal/audiostream"
	"listr/internal/song"
	"math"
	"net/http"
	"net/url"

	"github.com/google/uuid"
	"github.com/mjibson/go-dsp/fft"
)

type ShazamHandlerInterface interface {
	Init()
	SendMatchRequest(chunk *audiostream.Chunk) (*song.Song, error)
	Match(stream *audiostream.Stream) (*[]*song.Song, error) // Takes in audio stream
}

/*
SEARCH_FROM_FILE = (
        "https://amp.shazam.com/discovery/v5/{language}/{endpoint_country}/{device}/-/tag"
        "/{uuid_1}/{uuid_2}?sync=true&webv3=true&sampling=true"
        "&connected=&shazamapiversion=v3&sharehub=true&hubv5minorversion=v5.1&hidelb=true&video=v3"
    )
*/

type ShazamHandler struct {
	finds      *[]*song.Song
	requestURL *string
}

func (sh *ShazamHandler) Init() {
	reqURL := fmt.Sprintf(
		"https://amp.shazam.com/discovery/v5/en/US/desktop_mac/-/tag/%s/%s?sync=true&webv3=true&sampling=true&connected=&shazamapiversion=v3&sharehub=true&hubv5minorversion=v5.1&hidelb=true&video=v3",
		uuid.New().String(), uuid.New().String(),
	)

	findSlice := make([]*song.Song, 0, 5)
	sh.finds = &findSlice
	_, err := url.ParseRequestURI(reqURL)
	if err != nil {
		panic(err)
	}
	sh.requestURL = &reqURL
}

// ShazamResponse represents the response from the Shazam API
type ShazamResponse struct {
	Track struct {
		Title    string `json:"title"`
		Subtitle string `json:"subtitle"`
		Images   struct {
			CoverArt string `json:"coverart"`
		} `json:"images"`
	} `json:"track"`
}

func (sh *ShazamHandler) SendMatchRequest(c audiostream.Chunk) (*song.Song, error) {
	// Get audio data from chunk
	audioData := c.GetAudioData()
	if len(audioData) == 0 {
		return nil, fmt.Errorf("empty audio chunk")
	}

	// Convert raw bytes to PCM samples (16-bit mono)
	samples := make([]float64, len(audioData)/2)
	for i := 0; i < len(samples); i++ {
		// Convert 2 bytes to int16, then to float64
		sample := int16(audioData[i*2]) | (int16(audioData[i*2+1]) << 8)
		samples[i] = float64(sample) / 32768.0 // Normalize to [-1, 1]
	}

	// Apply FFT
	fftResult := fft.FFTReal(samples)

	// Find frequency peaks
	peaks := findFrequencyPeaks(fftResult, 16000) // Assuming 16kHz sample rate

	// Create signature from peaks
	signature := &audiostream.DecodedMessage{
		SampleRateHz:              16000,
		NumberSamples:             len(samples),
		FrequencyBandToSoundPeaks: make(map[audiostream.FrequencyBand][]audiostream.FrequencyPeak),
	}

	// Group peaks into frequency bands
	for _, peak := range peaks {
		band := getFrequencyBand(peak.Frequency)
		signature.FrequencyBandToSoundPeaks[band] = append(
			signature.FrequencyBandToSoundPeaks[band],
			audiostream.FrequencyPeak{
				FFTPassNumber:             peak.TimeIndex,
				PeakMagnitude:             peak.Magnitude,
				CorrectedPeakFrequencyBin: peak.FrequencyBin,
				SampleRateHz:              16000,
			},
		)
	}

	// Convert signature to URI format
	signatureURI, err := signature.EncodeToURI()
	if err != nil {
		return nil, fmt.Errorf("failed to encode signature: %v", err)
	}

	// Create request body
	requestBody := map[string]interface{}{
		"signature": map[string]interface{}{
			"uri": signatureURI,
		},
		"samplems": len(samples) * 1000 / 16000, // Convert samples to milliseconds
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %v", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", *sh.requestURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.114 Safari/537.36")

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Parse response
	var shazamResp ShazamResponse
	if err := json.NewDecoder(resp.Body).Decode(&shazamResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	// Create song object from response
	timestamp := c.GetTimestamp()
	title := shazamResp.Track.Title
	artist := shazamResp.Track.Subtitle

	return &song.Song{
		SongTitle:      &title,
		ArtistName:     &artist,
		TimestampFound: &timestamp,
	}, nil
}

// Peak represents a frequency peak in the audio
type Peak struct {
	Frequency    float64
	FrequencyBin int
	Magnitude    int
	TimeIndex    int
}

// findFrequencyPeaks finds significant peaks in the frequency domain
func findFrequencyPeaks(fftResult []complex128, sampleRate int) []Peak {
	const (
		minMagnitude = 1000 // Minimum magnitude to consider a peak
		windowSize   = 1024 // FFT window size
		hopSize      = 128  // Number of samples between windows
	)

	peaks := make([]Peak, 0)
	magnitudes := make([]float64, len(fftResult))

	// Calculate magnitudes
	for i, c := range fftResult {
		magnitudes[i] = math.Sqrt(real(c)*real(c) + imag(c)*imag(c))
	}

	// Find local maxima
	for i := 1; i < len(magnitudes)-1; i++ {
		if magnitudes[i] > minMagnitude &&
			magnitudes[i] > magnitudes[i-1] &&
			magnitudes[i] > magnitudes[i+1] {
			// Convert to frequency bin
			freqBin := i * sampleRate / windowSize
			// Convert to actual frequency
			freq := float64(freqBin) * float64(sampleRate) / float64(windowSize)

			peaks = append(peaks, Peak{
				Frequency:    freq,
				FrequencyBin: freqBin,
				Magnitude:    int(magnitudes[i]),
				TimeIndex:    i / hopSize,
			})
		}
	}

	return peaks
}

// getFrequencyBand determines which frequency band a peak belongs to
func getFrequencyBand(frequency float64) audiostream.FrequencyBand {
	switch {
	case frequency < 250:
		return audiostream.LowBand
	case frequency < 520:
		return audiostream.MidBand
	case frequency < 1450:
		return audiostream.HighBand
	default:
		return audiostream.VeryHighBand
	}
}
