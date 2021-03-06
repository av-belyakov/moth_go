package processingwebsocketrequest

import (
	"encoding/json"

	"moth_go/configure"
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
	IPAddress                    string `json:"ipAddress"`
	MaxCountProcessFiltering     int    `json:"maxCountProcessFiltering"`
	CountTransmissionInformation int    `json:"countTransmissionInformation"`
	VersionApp                   string `json:"versionApp"`
}

//MessageTypePong подтверждает настройки клиента
type MessageTypePong struct {
	MessageType string          `json:"messageType"`
	Info        InformationPong `json:"info"`
}

//RequestTypePing обрабатывает Ping запрос и отправляет ответ
func (messageTypePing *MessageTypePing) RequestTypePing(remoteIP string, mc *configure.MothConfig, accessClientsConfigure *configure.AccessClientsConfigure) ([]byte, error) {
	//записываем полученные от flashlight данные в AccessClientConfigure
	accessClientsConfigure.Addresses[remoteIP].CountTransmissionInformation = 0
	accessClientsConfigure.Addresses[remoteIP].MaxCountProcessFiltering = messageTypePing.Info.MaxCountProcessFiltering

	messageTypePong := MessageTypePong{
		MessageType: "pong",
		Info: InformationPong{
			IPAddress:                    mc.ExternalIPAddress,
			CountTransmissionInformation: 0,
			MaxCountProcessFiltering:     messageTypePing.Info.MaxCountProcessFiltering,
			VersionApp:                   mc.VersionApp,
		},
	}

	formatJSON, err := json.Marshal(messageTypePong)
	if err != nil {
		return nil, err
	}

	return formatJSON, err
}
