package processingWebsocketRequest

import (
	"encoding/json"
	"fmt"

	"moth_go/configure"
	"moth_go/saveMessageApp"
)

//MessageTypePingInfo описание вложенной информации для ping info
type MessageTypePingInfo struct {
	MaxCountProcessFiltering int `json:"maxCountProcessFiltering"`
}

//MessageTypePing содержит настройки для клиента
type MessageTypePing struct {
	Info MessageTypePingInfo `json:"info"`
}

//InformationPong хранит дополнительную информацию
type InformationPong struct {
	IPAddress                       string `json:"ipAddress"`
	MaxCountProcessFiltering        int    `json:"maxCountProcessFiltering"`
	CountTransmissionInformation    int    `json:"countTransmissionInformation"`
	IntervalTransmissionInformation int    `json:"intervalTransmissionInformation"`
}

//MessageTypePong подтверждает настройки клиента
type MessageTypePong struct {
	MessageType string          `json:"messageType"`
	Info        InformationPong `json:"info"`
}

//RequestTypePing обрабатывает Ping запрос и отправляет ответ
func (messageTypePing *MessageTypePing) RequestTypePing(remoteIP, externalIPAddress string, accessClientsConfigure *configure.AccessClientsConfigure, out chan<- []byte) {
	fmt.Println("\nPING: function RequestTypePing STARTING...")

	var messageTypePong MessageTypePong

	//записываем полученные от flashlight данные в AccessClientConfigure
	accessClientsConfigure.Addresses[remoteIP].CountTransmissionInformation = 0
	accessClientsConfigure.Addresses[remoteIP].MaxCountProcessFiltering = messageTypePing.Info.MaxCountProcessFiltering

	//формируем ответ
	messageTypePong.MessageType = "pong"
	messageTypePong.Info.IPAddress = externalIPAddress
	messageTypePong.Info.CountTransmissionInformation = 0
	messageTypePong.Info.MaxCountProcessFiltering = messageTypePing.Info.MaxCountProcessFiltering

	formatJSON, err := json.Marshal(messageTypePong)
	if err != nil {
		_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
		return
	}

	out <- formatJSON
}
