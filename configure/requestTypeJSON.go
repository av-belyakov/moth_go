package configure

/*
* Описания структур принимаемых JSON сообщений
* Версия 0.2, дата релиза 16.07.2018
* */

//MessageTypeSetting содержит детальную информацию о фильтрации
type MessageTypeSetting struct {
	DateTimeStart          uint64              `json:"dateTimeStart"`
	DateTimeEnd            uint64              `json:"dateTimeEnd"`
	IPAddress              []string            `json:"ipaddress"`
	Network                []string            `json:"network"`
	UseIndexes             bool                `json:"useIndexes"`
	CountFilesFiltering    int                 `json:"countFilesFiltering"`
	TotalNumberFilesFilter int                 `json:"totalNumberFilesFilter"`
	CountPartsIndexFiles   [2]int              `json:"countPartsIndexFiles"`
	ListFilesFilter        map[string][]string `json:"listFilesFilter"`
}

//MessageTypeFilterInfo содержит общую инфрмацию о фильтрации
type MessageTypeFilterInfo struct {
	Processing string             `json:"processing"`
	TaskIndex  string             `json:"taskIndex"`
	Settings   MessageTypeSetting `json:"settings"`
}

//MessageTypeFilter содержит всю информацию о выполянемой фильтрации
type MessageTypeFilter struct {
	Info MessageTypeFilterInfo `json:"info"`
}

//MessageTypeDownloadFilesInfo содержит подробную информацию о запросе на скачивание файлов
type MessageTypeDownloadFilesInfo struct {
	Processing                 string   `json:"processing"`
	TaskIndex                  string   `json:"taskIndex"`
	DownloadDirectoryFiles     string   `json:"downloadDirectoryFiles"`
	DownloadSelectedFiles      bool     `json:"downloadSelectedFiles"`
	CountDownloadSelectedFiles int      `json:"countDownloadSelectedFiles"`
	NumberMessageParts         [2]int   `json:"numberMessageParts"`
	ListDownloadSelectedFiles  []string `json:"listDownloadSelectedFiles"`
}

//MessageTypeDownloadFiles содержит запрос на скачивание файлов
type MessageTypeDownloadFiles struct {
	Info MessageTypeDownloadFilesInfo `json:"info"`
}
