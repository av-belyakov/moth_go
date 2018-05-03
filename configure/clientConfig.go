package configure

/*
* Описание структур и методов настроек получаемых от клиентов,
* в том числе и задач на фильтрацию сет. трафика, его выгрузку и т.д.
* Версия 0.11, дата релиза 03.05.2018
* */

import (
	"sync"

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
	CountTransmissionInformation int
	MaxCountProcessFiltering     int
	WsConnection                 *websocket.Conn
	mu                           sync.Mutex
	TaskFilter                   map[string]*InformationTaskFilter
}

/*
УБРАТЬ TaskFilter с информацией о выполняемых задачах,
соответственно удалить методы GetTaskFilter, GetCountTasksFilter, IsMaxCountProcessFiltering и
др. связанные с выполняемыми задачами по фильтрации
*/

//AccessClientsConfigure хранит представления с конфигурациями для клиентов
type AccessClientsConfigure struct {
	Addresses            map[string]*ClientsConfigure
	ChanInfoTranssmition chan []byte
}

//SendWsMessage используется для отправки сообщений через протокол websocket
func (clientsConfigure *ClientsConfigure) SendWsMessage(t int, v []byte) error {
	clientsConfigure.mu.Lock()
	defer clientsConfigure.mu.Unlock()

	return clientsConfigure.WsConnection.WriteMessage(t, v)
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

//IsMaxCountProcessFiltering проверяет количество одновременно выполняемых задач и возвращает true или false в зависимости от того, привышает ли максимальное количество задач для одного клиента
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
