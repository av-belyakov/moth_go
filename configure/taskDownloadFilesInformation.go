package configure

import (
	"fmt"
)

//FileInformationDownloadFiles содержит количество попыток передачи файла (от 3 до 0)
type FileInformationDownloadFiles struct {
	NumberTransferAttempts int
}

//FileInfoinQueue содержит подробную информацию о передаваемом файле
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
	//	StatusProcessTransmission string
	SelectedFiles          bool
	IsStoped               bool
	NumberPleasantMessages int
	ListDownloadFiles      map[string]*FileInformationDownloadFiles
}

//DownloadFilesInformation содержит информацию используемую для передачи файлов
type DownloadFilesInformation struct {
	RemoteIP map[string]*TaskInformationDownloadFiles
}

//CheckStatusProcessTransmission проверяет текущий статус задачи по передачи файлов
/*func (dfi *DownloadFilesInformation) CheckStatusProcessTransmission(remoteIP, carrentStatus string) bool {
	if ok := dfi.HasRemoteIPDownloadFiles(remoteIP); !ok {
		return false
	}

	return dfi.RemoteIP[remoteIP].StatusProcessTransmission == carrentStatus
}


//ChangeStatusProcessTransmission меняет статус передачи задачи по передачи файлов
func (dfi *DownloadFilesInformation) ChangeStatusProcessTransmission(remoteIP, newStatus string) bool {
	if ok := dfi.HasRemoteIPDownloadFiles(remoteIP); !ok {
		return false
	}

	dfi.RemoteIP[remoteIP].StatusProcessTransmission = newStatus

	return true
}
*/

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

//ChangeStopTaskDownloadFiles изменяет состояние свойства IsStoped
func (dfi *DownloadFilesInformation) ChangeStopTaskDownloadFiles(remoteIP, taskIndex string, status bool) {
	if dfi.HasTaskDownloadFiles(remoteIP, taskIndex) {
		dfi.RemoteIP[remoteIP].IsStoped = status
	}
}

//HasStopedTaskDownloadFiles проверяет состояние свойства IsStoped
func (dfi *DownloadFilesInformation) HasStopedTaskDownloadFiles(remoteIP, taskIndex string) bool {
	if dfi.HasTaskDownloadFiles(remoteIP, taskIndex) {
		return dfi.RemoteIP[remoteIP].IsStoped
	}

	return false
}

//AddTaskDownloadFiles добавляет новую задачу по выгрузке файлов
func (dfi *DownloadFilesInformation) AddTaskDownloadFiles(remoteIP string, infoTask *TaskInformationDownloadFiles) {
	dfi.RemoteIP = map[string]*TaskInformationDownloadFiles{
		remoteIP: infoTask,
	}
}

//DelTaskDownloadFiles удаляет задачу для выбранного ip адреса
func (dfi *DownloadFilesInformation) DelTaskDownloadFiles(remoteIP string) {

	fmt.Println("удаляем задачу по скачиванию файлов")

	delete(dfi.RemoteIP, remoteIP)
}

//RemoveFileFromListFiles удаляет указанный файл из списка DownloadFilesInformation.RemoteIP[<ip>].ListDownloadFiles
func (dfi *DownloadFilesInformation) RemoveFileFromListFiles(remoteIP, fileName string) {
	if ok := dfi.HasRemoteIPDownloadFiles(remoteIP); ok {
		delete(dfi.RemoteIP[remoteIP].ListDownloadFiles, fileName)
	}
}

//ClearListFiles очищает список файлов предназначенных для передачи
func (dfi *DownloadFilesInformation) ClearListFiles(remoteIP string) {
	if ok := dfi.HasRemoteIPDownloadFiles(remoteIP); ok {
		dfi.RemoteIP[remoteIP].ListDownloadFiles = map[string]*FileInformationDownloadFiles{}
	}
}
