package configure

//ChunkListParameters хранит набор параметров для разделения среза имен файлов на отдельные части
type ChunkListParameters struct {
	NumPart, CountParts, SizeChunk int
	ListFoundFiles                 []string
}
