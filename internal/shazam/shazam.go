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
}

/*
SEARCH_FROM_FILE = (
        "https://amp.shazam.com/discovery/v5/{language}/{endpoint_country}/{device}/-/tag"
        "/{uuid_1}/{uuid_2}?sync=true&webv3=true&sampling=true"
        "&connected=&shazamapiversion=v3&sharehub=true&hubv5minorversion=v5.1&hidelb=true&video=v3"
    )
*/

const (
	link = "https://amp.shazam.com/discovery/v5/en/US/desktop_mac/-/tag/962a6abf-e328-46da-bffe-0584f1e86a00/08f2e844-bf4a-47e5-b870-f2f3921a09ed?sync=true&webv3=true&sampling=true&connected=&shazamapiversion=v3&sharehub=true&hubv5minorversion=v5.1&hidelb=true&video=v3"
)

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

	return nil, nil
}
