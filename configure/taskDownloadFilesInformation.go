package configure

//FileInformationDownloadFiles содержит подробную информацию о передаваемом файле
type FileInformationDownloadFiles struct {
	FileName               string
	FileSize               int64
	FIleHash               string
	NumberTransferAttempts int
}

//TaskInformationDownloadFiles информация о выполняемой задаче
type TaskInformationDownloadFiles struct {
	TaskIndex          string
	DirectoryFiltering string
	FileInformation    *FileInformationDownloadFiles
}

//DownloadFilesInformation содержит информацию используемую для передачи файлов
type DownloadFilesInformation struct {
	RemoteIP map[string]*TaskInformationDownloadFiles
}

//HasTaskDownloadFiles проверяет наличие задачи по скачиванию файлов
func (dfi *DownloadFilesInformation) HasTaskDownloadFiles(remoteIP string) bool {
	_, fount := dfi.RemoteIP[remoteIP]
	return fount
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
