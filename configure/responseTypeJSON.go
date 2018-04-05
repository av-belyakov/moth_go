package configure

/*
* Описания структур принимаемых JSON сообщений
* Версия 0.1, дата релиза 19.03.2018
* */

//FilterinInfoPattern является шаблоном типа Info
type FilterinInfoPattern struct {
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

//MessageTypeFilteringStopInfo детальная информация при ОСТАНОВКИ выполнения фильтрации
type MessageTypeFilteringStopInfo struct {
	FilterinInfoPattern
}

//MessageTypeFilteringCompleteInfo детальная информация при ЗАВЕРШЕНИИ выполнения фильтрации
type MessageTypeFilteringCompleteInfo struct {
	FilterinInfoPattern
	CountCycleComplete int `json:"countCycleComplete"`
}

//MessageTypeFilteringStartInfoFirstPart детальная информаци, первый фрагмент (без имен файлов)
type MessageTypeFilteringStartInfoFirstPart struct {
	FilterinInfoPattern
	FilterCountPattern
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
	FilterinInfoPattern
	NumberMessageParts [2]int              `json:"numberMessageParts"`
	ListFilesFilter    map[string][]string `json:"listFilesFilter"`
}

//MessageTypeFilteringExecuteOrUnexecuteInfo детальная информация при выполнении или не выполнении фильтрации
type MessageTypeFilteringExecuteOrUnexecuteInfo struct {
	FilterinInfoPattern
	FilterCountPattern
	ProcessingFile InfoProcessingFile `json:"infoProcessingFile"`
	//	UnprocessingFile string `json:"unprocessingFile"`
}

/*
"messageType": "filtering",
                    "processing": "execute",
                    "clientIp": self.stringRequest.clientIp,
                    "taskIndex": self.taskIndex,
                    "countFilesFound": countFilesFound,
                    "countFilesProcessed": countFilesProcessed,
                    "countCycleComplete": countCycleComplete,
                    "countFoundFilesSize": countFoundFilesSize
*/
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

//MessageTypeFilteringComplete отправляется при завершении фильтрации
type MessageTypeFilteringComplete struct {
	MessageType string                           `json:"messageType"`
	Info        MessageTypeFilteringCompleteInfo `json:"info"`
}

//MessageTypeFilteringExecutedOrUnexecuted при выполнении или не выполнении фильтрации
type MessageTypeFilteringExecutedOrUnexecuted struct {
	MessageType string                                     `json:"messageType"`
	Info        MessageTypeFilteringExecuteOrUnexecuteInfo `json:"info"`
}
