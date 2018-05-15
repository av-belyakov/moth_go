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

//ClientsConfigure хранит информацию о конфигурации клиента
type ClientsConfigure struct {
	CountTransmissionInformation int
	MaxCountProcessFiltering     int
	mu                           sync.Mutex
	WsConnection                 *websocket.Conn
}

//ChanInfoFilterTask описание типа канала для передачи информации о фильтрации
type ChanInfoFilterTask struct {
	TaskIndex      string
	RemoteIP       string
	TypeProcessing string
}

//AccessClientsConfigure хранит представления с конфигурациями для клиентов
type AccessClientsConfigure struct {
	Addresses            map[string]*ClientsConfigure
	ChanInfoTranssmition chan []byte             //канал для передачи системной информации
	ChanInfoFilterTask   chan ChanInfoFilterTask //канал для передачи информации о выполняемой задачи по фильтрации сет. трафика
	ChanStopTaskFilter   chan string             //канал используемый для передачи идентификатора затачи с целью ее дальнейшей остановки
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
