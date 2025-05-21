package audiostream

import (
	"fmt"
	"net/url"
	"time"
)

type Chunk interface {
	Record(in chan byte) Chunk
}

type Stream interface {
	InitStream(V any)
	GetChunk() Chunk
}

type SoundCloudChunk struct {
	timestamp  *time.Duration
	audioChunk *[]byte
}

func (scc *SoundCloudChunk) Record(in chan byte) Chunk {
	var newChunk SoundCloudChunk
	ts := 0 * time.Second
	newChunk.timestamp = &ts

	var chunkBuffer []byte
	for i := 0; i < 5; i++ {
		buf := <-in
		chunkBuffer = append(chunkBuffer, buf)
	}
	scc.audioChunk = &chunkBuffer
	return &newChunk
}

type SoundCloudStream struct {
	url          *string
	chunkCounter int
}

func (scs *SoundCloudStream) InitStream(link *string) {
	_, err := url.ParseRequestURI(*link)
	if err != nil {
		panic(err)
	}
	scs.url = link
	scs.chunkCounter = 0
}

func (scs *SoundCloudStream) GetChunk() {
	//Stream in next 10 second chunk of audio

	fmt.Printf("Streaming chunk %d from %s", scs.chunkCounter, *scs.url)
	scs.chunkCounter++
}
