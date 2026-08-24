package iterator

type Iterator interface {
	SeekToFirst()
	Seek(key []byte)
	Next()
	Valid() bool
	Key() []byte
	Value() []byte
	Close()
}
