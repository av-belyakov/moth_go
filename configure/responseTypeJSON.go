package configure

/*
* Описания структур отправляемых JSON сообщений
* Версия 0.11, дата релиза 13.06.2018
* */

//FilterInfoPattern является шаблоном типа Info
type FilterInfoPattern struct {
	Processing string `json:"processing"`
	TaskIndex  string `json:"taskIndex"`
	IPAddress  string `json:"ipAddress"`
}

//FilterCountPattern шаблон для частей учета некоторого количества
type FilterCountPattern struct {
	CountCycleComplete    int   `json:"countCycleComplete"`
	CountFilesFound       int   `json:"countFilesFound"`
	CountFoundFilesSize   int64 `json:"countFoundFilesSize"`
	CountFilesProcessed   int   `json:"countFilesProcessed"`
	CountFilesUnprocessed int   `json:"countFilesUnprocessed"`
}

//InfoProcessingFile информация об обработанном файле
type InfoProcessingFile struct {
	FileName          string `json:"fileName"`
	DirectoryLocation string `json:"directoryLocation"`
	StatusProcessed   bool   `json:"statusProcessed"`
}

//MessageTypeFilteringStopInfo сообщение при ОСТАНОВ выполнения фильтрации
type MessageTypeFilteringStopInfo struct {
	FilterInfoPattern
}

//MessageTypeFilteringCompleteInfoFirstPart детальная информация при ЗАВЕРШЕНИИ выполнения фильтрации (первая часть)
type MessageTypeFilteringCompleteInfoFirstPart struct {
	FilterInfoPattern
	FilterCountPattern
	NumberMessageParts [2]int `json:"numberMessageParts"`
}

//MessageTypeFilteringCompleteInfoSecondPart информация при ЗАВЕРШЕНИИ выполнения фильтрации (вторая часть)
type MessageTypeFilteringCompleteInfoSecondPart struct {
	FilterInfoPattern
	NumberMessageParts            [2]int           `json:"numberMessageParts"`
	ListFilesFoundDuringFiltering []FoundFilesInfo `json:"listFilesFoundDuringFiltering"`
}

//MessageTypeFilteringStartInfoFirstPart детальная информаци, первый фрагмент (без имен файлов)
type MessageTypeFilteringStartInfoFirstPart struct {
	FilterInfoPattern
	DirectoryFiltering      string         `json:"directoryFiltering"`
	CountDirectoryFiltering int            `json:"countDirectoryFiltering"`
	CountFullCycle          int            `json:"countFullCycle"`
	CountFilesFiltering     int            `json:"countFilesFiltering"`
	CountMaxFilesSize       int64          `json:"countMaxFilesSize"`
	UseIndexes              bool           `json:"useIndexes"`
	NumberMessageParts      [2]int         `json:"numberMessageParts"`
	ListCountFilesFilter    map[string]int `json:"listCountFilesFilter"`
}

//MessageTypeFilteringStartInfoSecondPart детальная информация с именами файлов
type MessageTypeFilteringStartInfoSecondPart struct {
	FilterInfoPattern
	UseIndexes         bool                `json:"useIndexes"`
	NumberMessageParts [2]int              `json:"numberMessageParts"`
	ListFilesFilter    map[string][]string `json:"listFilesFilter"`
}

//MessageTypeFilteringExecuteOrUnexecuteInfo детальная информация при выполнении или не выполнении фильтрации
type MessageTypeFilteringExecuteOrUnexecuteInfo struct {
	FilterInfoPattern
	FilterCountPattern
	InfoProcessingFile `json:"infoProcessingFile"`
}

//MessageTypeFilteringStartFirstPart при начале фильтрации (первая часть)
type MessageTypeFilteringStartFirstPart struct {
	MessageType string                                 `json:"messageType"`
	Info        MessageTypeFilteringStartInfoFirstPart `json:"info"`
}

//MessageTypeFilteringStartSecondPart при начале фильтрации (первая часть)
type MessageTypeFilteringStartSecondPart struct {
	MessageType string                                  `json:"messageType"`
	Info        MessageTypeFilteringStartInfoSecondPart `json:"info"`
}

//MessageTypeFilteringStop отправляется для подтверждения остановки фильтрации
type MessageTypeFilteringStop struct {
	MessageType string                       `json:"messageType"`
	Info        MessageTypeFilteringStopInfo `json:"info"`
}

//MessageTypeFilteringCompleteFirstPart отправляется при завершении фильтрации
type MessageTypeFilteringCompleteFirstPart struct {
	MessageType string                                    `json:"messageType"`
	Info        MessageTypeFilteringCompleteInfoFirstPart `json:"info"`
}

//MessageTypeFilteringCompleteSecondPart отправляется при завершении фильтрации
type MessageTypeFilteringCompleteSecondPart struct {
	MessageType string                                     `json:"messageType"`
	Info        MessageTypeFilteringCompleteInfoSecondPart `json:"info"`
}

//MessageTypeFilteringExecutedOrUnexecuted при выполнении или не выполнении фильтрации
type MessageTypeFilteringExecutedOrUnexecuted struct {
	MessageType string                                     `json:"messageType"`
	Info        MessageTypeFilteringExecuteOrUnexecuteInfo `json:"info"`
}

//MessageTypeDownloadFilesInfoExecute содержит информацию передоваемую при сообщениях о передаче информации о файле
type MessageTypeDownloadFilesInfoExecute struct {
	MessageTypeDownloadFilesInfoReadyOrFinished
	FileName string `json:"fileName"`
	FileSize string `json:"fileSize"`
	FileHash string `json:"fileHash"`
}

//MessageTypeDownloadFilesInfoReadyOrFinished содержит информацию передоваемую при сообщениях о готовности или завершении передачи
type MessageTypeDownloadFilesInfoReadyOrFinished struct {
	Processing string `json:"processing"`
	TaskIndex  string `json:"taskIndex"`
}

//MessageTypeDownloadFilesReadyOrFinished применяется для отправки сообщений о готовности или завершении передачи
type MessageTypeDownloadFilesReadyOrFinished struct {
	MessageType string
	Info        MessageTypeDownloadFilesInfoReadyOrFinished `json:"info"`
}

//MessageTypeDownloadFilesExecute применяется для отправки сообщений о передаче файлов
type MessageTypeDownloadFilesExecute struct {
	MessageType string
	Info        MessageTypeDownloadFilesInfoExecute `json:"info"`
}
