package configure

/*
Описание пользовательского типа с информацией о выполняемых задачах по фильтрации
сет. трафика, а также описание типа канала для передачи информации по результатам фильтрации
Версия 0.1, дата релиза 03.05.2018
*/

//InfoFilterSettings хранит информацию о параметрах используемых для фильтрации
type InfoFilterSettings struct {
	DateTimeStart, DateTimeEnd uint64
	IPAddress, Network         []string
	UserInput                  string
}

//TaskInformation хранит подробную информацию о выполняемой задачи по фильтрации сет. трафика
type TaskInformation struct {
	RemoteIP                string
	FilterSettings          *InfoFilterSettings
	UseIndexes              bool
	DirectoryFiltering      string
	ProcessingFileName      string
	StatusProcessedFile     bool
	CountDirectoryFiltering int
	CountFullCycle          int
	CountCycleComplete      int
	CountFilesFiltering     int
	CountFilesFound         int
	CountFilesProcessed     int
	CountFilesUnprocessed   int
	CountMaxFilesSize       int64
	CountFoundFilesSize     int64
	ListFilesFilter         map[string][]string
	TypeProcessing          string
}

//InformationFilteringTask хранит информацию о выполняющихся задачах по фильтрации сет. трафика
type InformationFilteringTask struct {
	TaskID map[string]*TaskInformation
}

//ChanInfoFilterTask описание типа канала для передачи информации о фильтрации
type ChanInfoFilterTask struct {
	TaskIndex           string
	RemoteIP            string
	TypeProcessing      string
	ProcessingFileName  string
	StatusProcessedFile bool
	CountFilesFound     int
	CountFoundFilesSize int64
}

//GetInfoTaskFilter поиск задач фильтрации по IP клиента
func (ift *InformationFilteringTask) GetInfoTaskFilter(remoteIP string) (taskIndex string, taskInformation *TaskInformation) {
	for taskIndex, task := range ift.TaskID {
		if task.RemoteIP == remoteIP {
			return taskIndex, task
		}
	}
	return
}

//GetCountTasksFilterForClient возвращает количество выполняемых для указанного клиента задач по фильтрации
func (ift *InformationFilteringTask) GetCountTasksFilterForClient(remoteIP string) int {
	countProcessTasks := 0

	for _, task := range ift.TaskID {
		if task.RemoteIP == remoteIP {
			countProcessTasks++
		}
	}

	return countProcessTasks
}

//IsMaxConcurrentProcessFiltering проверяет количество одновременно выполняемых задач и возвращает true или false в зависимости от того, привышает ли максимальное количество задач для одного клиента
func (ift *InformationFilteringTask) IsMaxConcurrentProcessFiltering(remoteIP string, concurrent int) bool {
	countProcessTasks := 0

	for _, task := range ift.TaskID {
		if task.RemoteIP == remoteIP {
			countProcessTasks++
		}
	}

	return countProcessTasks > concurrent
}
