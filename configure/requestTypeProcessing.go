package configure

//ParametrsFunctionRequestFilter параметры для передачи функции обработки фильтрации
type ParametrsFunctionRequestFilter struct {
	RemoteIP, ExternalIP   string
	PathStorageFilterFiles string
	AccessClientsConfigure *AccessClientsConfigure
	CurrentDisks           []string
	TypeAreaNetwork        int
	ChanStopTaskFilter     chan string
}
