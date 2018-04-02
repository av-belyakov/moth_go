package configure

/*
* Описание структур и методов настроек получаемых от клиентов,
* в том числе и задач на фильтрацию сет. трафика, его выгрузку и т.д.
* Версия 0.1, дата релиза 19.02.2018
* */

import (
	"github.com/gorilla/websocket"
)

//InformationTaskFilter содержит всю информацию по выполняемым заданиям фильтрации
type InformationTaskFilter struct {
	DateTimeStart, DateTimeEnd uint64
	IPAddress, Network         []string
	UseIndexes                 bool
	DirectoryFiltering         string
	CountDirectoryFiltering    int
	CountFullCycle             int
	CountCycleComplete         int
	CountFilesFiltering        int
	CountFilesFound            int
	CountFilesProcessed        int
	CountFilesUnprocessed      int
	CountMaxFilesSize          int64
	CountFoundFilesSize        int64
	ListFilesFilter            map[string][]string
}

//ClientsConfigure хранит информацию о конфигурации клиента
type ClientsConfigure struct {
	CountTransmissionInformation    int
	IntervalTransmissionInformation int
	MaxCountProcessFiltering        int
	WsConnection                    *websocket.Conn
	TaskFilter                      map[string]*InformationTaskFilter
}

//AccessClientsConfigure хранит представления с конфигурациями для клиентов
type AccessClientsConfigure struct {
	Addresses map[string]*ClientsConfigure
}

//IPAddressIsExist поиск ip адреса в срезе AccessIPAddress
func (a *AccessClientsConfigure) IPAddressIsExist(ipaddress string) bool {
	_, found := a.Addresses[ipaddress]
	return found
}

//GetTaskFilter поиск задач фильтрации по IP клиента и идентификатору задачи
func (a *AccessClientsConfigure) GetTaskFilter(remoteIP, taskFilter string) *InformationTaskFilter {
	return a.Addresses[remoteIP].TaskFilter[taskFilter]
}

//GetCountTasksFilter возвращает количество выполняемых для указанного клиента задач по фильтрации
func (a *AccessClientsConfigure) GetCountTasksFilter(remoteIP string) int {
	return len(a.Addresses[remoteIP].TaskFilter)
}

//IsMaxCountProcessFiltering проверяет количество одновременно выполняемых задач и возвращает true или false в ззависимости от того привышает ли максимальное количество задач
func (a *AccessClientsConfigure) IsMaxCountProcessFiltering(remoteIP string) bool {
	return (len(a.Addresses[remoteIP].TaskFilter) > a.Addresses[remoteIP].MaxCountProcessFiltering)
}

//GetCountDirectoryFiltering получить количество директорий в которых обрабатывается файлы
func (a *AccessClientsConfigure) GetCountDirectoryFiltering(remoteIP, taskIndex string) int {
	var num int
	for _, value := range a.Addresses[remoteIP].TaskFilter[taskIndex].ListFilesFilter {
		if len(value) > 0 {
			num++
		}
	}
	return num
}

//GetCountFullFilesFiltering общее количество файлов необходимых для выполнения фильтрации
func (a *AccessClientsConfigure) GetCountFullFilesFiltering(remoteIP, taskIndex string) int {
	var num int
	for _, value := range a.Addresses[remoteIP].TaskFilter[taskIndex].ListFilesFilter {
		num += len(value)
	}
	return num
}
