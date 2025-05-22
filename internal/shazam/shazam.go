package shazam

import (
	"fmt"
	"listr/internal/audiostream"
	"listr/internal/song"
	"net/url"

	"github.com/google/uuid"
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

func (sh *ShazamHandler) SendMatchRequest(c audiostream.Chunk) (*song.Song, error) {
	// Get audio data from chunk
	audioData := c.GetAudioData()
	if len(audioData) == 0 {
		return nil, fmt.Errorf("empty audio chunk")
	}

	// TODO: Implement frequency peak detection from audio data

	// For now, return a placeholder song
	timestamp := c.GetTimestamp()
	return &song.Song{
		SongTitle:      nil,
		ArtistName:     nil,
		AlbumName:      nil,
		TimestampFound: &timestamp,
	}, nil
}
