package song

import "time"

type Song struct {
	SongTitle      *string
	ArtistName     *string
	AlbumName      *string
	TimestampFound *time.Duration
	//Album Art Link?
}
