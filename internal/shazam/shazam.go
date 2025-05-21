package shazam

type ShazamHandler interface {
	SendRecognizeRequest(chunk Chunk)
}
