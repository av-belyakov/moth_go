package helpers

import "moth_go/configure"

//GetChunkListFilesFoundDuringFiltering делит срез имен файлов на отдельные части
func GetChunkListFilesFoundDuringFiltering(clp configure.ChunkListParameters) []configure.FoundFilesInfo {
	listFilesFilter := []configure.FoundFilesInfo{}

	if clp.NumPart == 1 {
		if len(clp.ListFoundFiles) < clp.SizeChunk {
			listFilesFilter = clp.ListFoundFiles[:]
		} else {
			listFilesFilter = clp.ListFoundFiles[:clp.SizeChunk]
		}
	} else {
		num := clp.SizeChunk * (clp.NumPart - 1)
		numEnd := num + clp.SizeChunk

		if (clp.NumPart == clp.CountParts) && (num < len(clp.ListFoundFiles)) {
			listFilesFilter = clp.ListFoundFiles[num:]
		}
		if (clp.NumPart < clp.CountParts) && (numEnd < len(clp.ListFoundFiles)) {
			listFilesFilter = clp.ListFoundFiles[num:numEnd]
		}

	}

	return listFilesFilter
}
