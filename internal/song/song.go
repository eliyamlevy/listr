package song

import "time"

type Song struct {
	SongTitle      *string
	ArtistName     *string
	TimestampFound *time.Duration
	//Album Art Link?
}
