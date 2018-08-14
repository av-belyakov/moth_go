package configure

//FileInformationDownloadFiles содержит подробную информацию о передаваемом файле
type FileInformationDownloadFiles struct {
	NumberTransferAttempts int
}

//FileInfoinQueue информация по файлы в очереди
type FileInfoinQueue struct {
	FileName string
	FileHash string
	FileSize int64
}

//TaskInformationDownloadFiles информация о выполняемой задаче
type TaskInformationDownloadFiles struct {
	TaskIndex               string
	DirectoryFiltering      string
	TotalCountDownloadFiles int
	FileInQueue             FileInfoinQueue
	SelectedFiles           bool
	NumberPleasantMessages  int
	ListDownloadFiles       map[string]*FileInformationDownloadFiles
}

//DownloadFilesInformation содержит информацию используемую для передачи файлов
type DownloadFilesInformation struct {
	RemoteIP map[string]*TaskInformationDownloadFiles
}

//HasRemoteIPDownloadFiles проверяет наличие ip адреса с которого выполнялись запросы по скачиванию файлов
func (dfi *DownloadFilesInformation) HasRemoteIPDownloadFiles(remoteIP string) bool {
	_, fount := dfi.RemoteIP[remoteIP]
	return fount
}

//HasTaskDownloadFiles проверяет наличие задачи по скачиванию файлов
func (dfi *DownloadFilesInformation) HasTaskDownloadFiles(remoteIP, taskIndex string) bool {
	if ok := dfi.HasRemoteIPDownloadFiles(remoteIP); !ok {
		return false
	}

	return dfi.RemoteIP[remoteIP].TaskIndex == taskIndex
}

//AddTaskDownloadFiles добавляет новую задачу по выгрузке файлов
func (dfi *DownloadFilesInformation) AddTaskDownloadFiles(remoteIP string, infoTask *TaskInformationDownloadFiles) {
	dfi.RemoteIP = map[string]*TaskInformationDownloadFiles{
		remoteIP: infoTask,
	}
}

//DelTaskDownloadFiles удаляет задачу для выбранного ip адреса
func (dfi *DownloadFilesInformation) DelTaskDownloadFiles(remoteIP string) {
	delete(dfi.RemoteIP, remoteIP)
}
