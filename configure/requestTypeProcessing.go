package configure

//ParametrsFunctionRequestFilter параметры для передачи функции обработки фильтрации
type ParametrsFunctionRequestFilter struct {
	RemoteIP               string
	ExternalIP             string
	PathStorageFilterFiles string
	AccessClientsConfigure *AccessClientsConfigure
	CurrentDisks           []string
	TypeAreaNetwork        int
	ChanStopTaskFilter     chan string
}

//ParametrsFunctionRequestDownloadFiles параметры для передачи функции обработки выгрузки файлов
type ParametrsFunctionRequestDownloadFiles struct {
	RemoteIP               string
	ExternalIP             string
	PathStorageFilterFiles string
	AccessClientsConfigure *AccessClientsConfigure
}
