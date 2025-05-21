package audiostream

type Chunk interface {
	Record(in ch) Chunk
}
