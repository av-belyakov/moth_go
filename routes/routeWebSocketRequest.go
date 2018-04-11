package routes

import (
	"encoding/json"
	"fmt"
	"time"

	"moth_go/configure"
	"moth_go/processingWebsocketRequest"
	"moth_go/saveMessageApp"
	"moth_go/sysInfo"
)

//MessageType содержит тип JSON сообщения
type MessageType struct {
	Type string `json:"messageType"`
}

var messageType MessageType
var messageTypePing processingWebsocketRequest.MessageTypePing
var messageTypeFilter configure.MessageTypeFilter

func actionConnectionBroken(accessClientsConfigure *configure.AccessClientsConfigure, remoteIP string) {
	//удаляем информацию о конфигурации для данного IP
	delete(accessClientsConfigure.Addresses, remoteIP)
}

//RouteWebSocketRequest маршрутизирует запросы
func RouteWebSocketRequest(remoteIP string, accessClientsConfigure *configure.AccessClientsConfigure, mc *configure.MothConfig) {
	fmt.Println("*** RouteWebSocketRequest.go ***")

	c := accessClientsConfigure.Addresses[remoteIP].WsConnection

	chanTypePing := make(chan []byte)
	chanInfoTranssmition := make(chan []byte)

	defer func() {
		fmt.Println("closed channels")
		close(chanTypePing)
		//close(chanInfoTranssmition)
	}()

	var parametrsFunctionRequestFilter configure.ParametrsFunctionRequestFilter

	for {
		_, message, err := c.ReadMessage()
		if err != nil {
			_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
			break
		}
		if err = json.Unmarshal(message, &messageType); err != nil {
			_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
		}

		switch messageType.Type {
		case "ping":
			fmt.Println("routing to PING...")

			if err = json.Unmarshal(message, &messageTypePing); err != nil {
				_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
			}

			go messageTypePing.RequestTypePing(remoteIP, mc.ExternalIPAddress, accessClientsConfigure, chanTypePing)

			err = c.WriteMessage(1, <-chanTypePing)
			if err != nil {
				_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
			}

			timer := accessClientsConfigure.Addresses[remoteIP].IntervalTransmissionInformation
			ticker := time.NewTicker(time.Duration(timer) * time.Second)
			defer func() {
				fmt.Println("closed ticker channel")
				ticker.Stop()
			}()

			go func() {
				for {
					select {
					case <-ticker.C:
						//fmt.Println("\nCurrent time: ", t)

						go sysInfo.GetSystemInformation(chanInfoTranssmition, mc)
					}
				}
			}()

			go func() {
				for {
					messageResponse := <-chanInfoTranssmition

					//fmt.Println("\nSENDING SOURCE INFO TO -----> Flashlight")

					err = c.WriteMessage(1, messageResponse)
					if err != nil {
						fmt.Println(err)
						_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))

						//чистим конфигурацию и закрываем канал ticker
						actionConnectionBroken(accessClientsConfigure, remoteIP)
						return
					}
				}
			}()
		case "filtering":
			fmt.Println("routing to FILTERING...")

			if err = json.Unmarshal(message, &messageTypeFilter); err != nil {
				_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
			}

			parametrsFunctionRequestFilter.RemoteIP = remoteIP
			parametrsFunctionRequestFilter.ExternalIP = mc.ExternalIPAddress
			parametrsFunctionRequestFilter.CurrentDisks = mc.CurrentDisks
			parametrsFunctionRequestFilter.PathStorageFilterFiles = mc.PathStorageFilterFiles
			parametrsFunctionRequestFilter.TypeAreaNetwork = mc.TypeAreaNetwork
			parametrsFunctionRequestFilter.AccessClientsConfigure = accessClientsConfigure

			go processingWebsocketRequest.RequestTypeFilter(&parametrsFunctionRequestFilter, &messageTypeFilter)

		case "download files":
			fmt.Println("routing to DOWNLOAD FILES...")

		}
	}
}
