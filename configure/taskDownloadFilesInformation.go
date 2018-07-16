package configure

//FileInformation содержит подробную информацию о передаваемом файле
type FileInformation struct {
	FileName               string
	FileSize               int64
	FIleHash               string
	NumberTransferAttempts int
}

//DownloadFilesInformation содержит информацию используемую для передачи файлов
type DownloadFilesInformation struct {
	RemoteIP           string
	TaskIndex          string
	DirectoryFiltering string
	FileInformation    *FileInformation
}
