package audiostream

import (
	"fmt"
	"net/url"
	"time"
)

// Chunk represents a segment of audio data with its position in the stream
type Chunk interface {
	// Record captures audio data from the input channel into this chunk
	Record(in chan byte) Chunk
	// GetAudioData returns the raw audio data for this chunk
	GetAudioData() []byte
	// GetTimestamp returns the start time of this chunk in the stream
	GetTimestamp() time.Duration
	// GetDuration returns the duration of this chunk
	GetDuration() time.Duration
}

type Stream interface {
	InitStream(V any) error
	GetChunk() (Chunk, error)
}

// SoundCloudChunk represents a 10-second segment of audio from a SoundCloud stream
type SoundCloudChunk struct {
	timestamp  *time.Duration // Start time of this chunk in the stream
	audioChunk *[]byte        // Raw audio data
}

// Record captures audio data from the input channel into this chunk
func (scc *SoundCloudChunk) Record(in chan byte) Chunk {
	var newChunk SoundCloudChunk
	newTimestamp := *scc.timestamp + 10*time.Second // Each chunk is 10 seconds
	newChunk.timestamp = &newTimestamp

	// Read 10 seconds of audio data (assuming 16kHz, 16-bit mono)
	// 10 seconds * 16000 samples/second * 2 bytes/sample = 320,000 bytes
	chunkBuffer := make([]byte, 320000)
readLoop:
	for i := 0; i < len(chunkBuffer); i++ {
		select {
		case buf, ok := <-in:
			if !ok {
				// Channel closed, return partial chunk
				chunkBuffer = chunkBuffer[:i]
				break readLoop
			}
			chunkBuffer[i] = buf
		case <-time.After(100 * time.Millisecond):
			// Timeout, return partial chunk
			chunkBuffer = chunkBuffer[:i]
			break readLoop
		}
	}

	scc.audioChunk = &chunkBuffer
	return &newChunk
}

// GetAudioData returns the raw audio data for this chunk
func (scc *SoundCloudChunk) GetAudioData() []byte {
	return *scc.audioChunk
}

// GetTimestamp returns the start time of this chunk in the stream
func (scc *SoundCloudChunk) GetTimestamp() time.Duration {
	return *scc.timestamp
}

// GetDuration returns the duration of this chunk
// For a full chunk, this will be 10 seconds. For partial chunks (due to stream end or timeout),
// this will be calculated based on the actual amount of audio data.
func (scc *SoundCloudChunk) GetDuration() time.Duration {
	// Calculate duration based on actual audio data size
	// For 16kHz, 16-bit mono: 1 second = 32000 bytes
	bytesPerSecond := 32000
	return time.Duration(len(*scc.audioChunk)/bytesPerSecond) * time.Second
}

type SoundCloudStream struct {
	url          string
	chunkCounter int
	audioChan    chan byte
}

func (scs *SoundCloudStream) InitStream(link any) error {
	urlStr, ok := link.(string)
	if !ok {
		return fmt.Errorf("expected string URL, got %T", link)
	}

	_, err := url.ParseRequestURI(urlStr)
	if err != nil {
		return fmt.Errorf("invalid URL: %v", err)
	}

	scs.url = urlStr
	scs.chunkCounter = 0
	scs.audioChan = make(chan byte, 320000) // Buffer for one chunk

	// Start streaming in a goroutine
	go scs.streamAudio()
	return nil
}

func (scs *SoundCloudStream) GetChunk() (Chunk, error) {
	if scs.audioChan == nil {
		return nil, fmt.Errorf("stream not initialized")
	}

	timestamp := time.Duration(scs.chunkCounter*10) * time.Second
	chunk := &SoundCloudChunk{
		timestamp: &timestamp,
	}

	// Record the next chunk of audio
	newChunk := chunk.Record(scs.audioChan)
	scs.chunkCounter++

	return newChunk, nil
}

func (scs *SoundCloudStream) streamAudio() {
	// TODO: Implement actual SoundCloud streaming
	// For now, just simulate streaming by sending some test data
	for {
		select {
		case scs.audioChan <- byte(scs.chunkCounter % 256):
			// Simulate streaming by sending some test data
		case <-time.After(100 * time.Millisecond):
			// Simulate network delay
		}
	}
}
