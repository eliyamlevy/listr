package shazam

import (
	"listr/internal/audiostream"
	"listr/internal/song"
)

type ShazamHandler interface {
	SendRecognizeRequest(chunk audiostream.Chunk) (song.Song, error)
}

/*
SEARCH_FROM_FILE = (
        "https://amp.shazam.com/discovery/v5/{language}/{endpoint_country}/{device}/-/tag"
        "/{uuid_1}/{uuid_2}?sync=true&webv3=true&sampling=true"
        "&connected=&shazamapiversion=v3&sharehub=true&hubv5minorversion=v5.1&hidelb=true&video=v3"
    )
*/
