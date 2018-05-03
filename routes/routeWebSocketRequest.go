package routes

import (
	"encoding/json"
	"fmt"

	"moth_go/configure"
	"moth_go/processingWebsocketRequest"
	"moth_go/saveMessageApp"
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

//sendFilterTaskInfo отправляет сообщение о выполняемой или выполненной задаче по фильтрации сет. трафика
func sendFilterTaskInfoAfterPingMessage(remoteIP, ExternalIP string, accessClientsConfigure *configure.AccessClientsConfigure, ift *configure.InformationFilteringTask) {
	for taskIndex, task := range ift.TaskID {
		if task.RemoteIP == remoteIP {
			//формируем сообщение о ходе фильтрации
			var mtfeou configure.MessageTypeFilteringExecutedOrUnexecuted

			mtfeou.MessageType = "filtering"
			mtfeou.Info.IPAddress = remoteIP
			mtfeou.Info.TaskIndex = taskIndex
			mtfeou.Info.Processing = task.TypeProcessing
			mtfeou.Info.ProcessingFile.FileName = task.ProcessingFileName
			mtfeou.Info.ProcessingFile.DirectoryLocation = task.DirectoryFiltering
			mtfeou.Info.CountFilesProcessed = task.CountFilesProcessed
			mtfeou.Info.CountFilesUnprocessed = task.CountFilesUnprocessed
			mtfeou.Info.ProcessingFile.StatusProcessed = task.StatusProcessedFile
			mtfeou.Info.CountCycleComplete = task.CountCycleComplete
			mtfeou.Info.CountFilesFound = task.CountFilesFound
			mtfeou.Info.CountFoundFilesSize = task.CountFoundFilesSize

			formatJSON, err := json.Marshal(&mtfeou)
			if err != nil {
				_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
			}

			if err := accessClientsConfigure.Addresses[remoteIP].SendWsMessage(1, formatJSON); err != nil {
				_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
			}
		}
	}
}

//RouteWebSocketRequest маршрутизирует запросы
func RouteWebSocketRequest(remoteIP string, accessClientsConfigure *configure.AccessClientsConfigure, ift *configure.InformationFilteringTask, mc *configure.MothConfig) {
	fmt.Println("*** RouteWebSocketRequest.go ***")

	c := accessClientsConfigure.Addresses[remoteIP].WsConnection

	chanTypePing := make(chan []byte)

	defer close(chanTypePing)

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

			//отправляем сообщение о выполняемой или выполненной задачи по фильтрации (если соединение было разорванно и вновь установленно)
			sendFilterTaskInfoAfterPingMessage(remoteIP, mc.ExternalIPAddress, accessClientsConfigure, ift)

			go func() {
				for {
					messageResponse := <-accessClientsConfigure.ChanInfoTranssmition

					//fmt.Println("\nSENDING SOURCE INFO TO -----> Flashlight")

					err = c.WriteMessage(1, messageResponse)
					if err != nil {
						fmt.Println(err)
						_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))

						//чистим конфигурацию
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

			go processingWebsocketRequest.RequestTypeFilter(&parametrsFunctionRequestFilter, &messageTypeFilter, ift)

			/*
			   Написать go подпрограмму для приема результатов фильтрации
			   из канала типа ChanInfoFilterTask
			*/

		case "download files":
			fmt.Println("routing to DOWNLOAD FILES...")

		}
	}
}
