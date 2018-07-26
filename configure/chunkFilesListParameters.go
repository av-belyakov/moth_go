package configure

//FoundFilesInfo содержит информацию о файле
type FoundFilesInfo struct {
	FileName string `json:"fileName"`
	FileSize int64  `json:"fileSize"`
}

//ChunkListParameters хранит набор параметров для разделения среза имен файлов на отдельные части
type ChunkListParameters struct {
	NumPart, CountParts, SizeChunk int
	ListFoundFiles                 []FoundFilesInfo
}
