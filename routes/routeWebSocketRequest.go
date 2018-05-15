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

//sendFilterTaskInfo отправляет сообщение о выполняемой или выполненной задаче по фильтрации сет. трафика
func sendFilterTaskInfoAfterPingMessage(remoteIP, ExternalIP string, acc *configure.AccessClientsConfigure, ift *configure.InformationFilteringTask) {
	for taskIndex, task := range ift.TaskID {
		fmt.Println("FROM TASKS, received filtering task ID:", taskIndex)

		if task.RemoteIP == remoteIP {
			fmt.Println("FROM TASKS", task.RemoteIP, " == ", remoteIP)

			if sourceData, ok := acc.Addresses[task.RemoteIP]; ok {
				switch task.TypeProcessing {
				case "execute":
					var mtfeou configure.MessageTypeFilteringExecutedOrUnexecuted
					mtfeou.MessageType = "filtering"
					mtfeou.Info.IPAddress = task.RemoteIP
					mtfeou.Info.TaskIndex = taskIndex
					mtfeou.Info.Processing = task.TypeProcessing
					mtfeou.Info.ProcessingFile.FileName = task.ProcessingFileName
					mtfeou.Info.ProcessingFile.DirectoryLocation = task.DirectoryFiltering
					mtfeou.Info.ProcessingFile.StatusProcessed = task.StatusProcessedFile
					mtfeou.Info.CountFilesProcessed = task.CountFilesProcessed
					mtfeou.Info.CountFilesUnprocessed = task.CountFilesUnprocessed
					mtfeou.Info.CountCycleComplete = task.CountCycleComplete
					mtfeou.Info.CountFilesFound = task.CountFilesFound
					mtfeou.Info.CountFoundFilesSize = task.CountFoundFilesSize

					formatJSON, err := json.Marshal(&mtfeou)
					if err != nil {
						_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
					}

					if err := sourceData.SendWsMessage(1, formatJSON); err != nil {
						_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
					}
				case "complete":
					messageTypeFilteringComplete := configure.MessageTypeFilteringComplete{
						MessageType: "filtering",
						Info: configure.MessageTypeFilteringCompleteInfo{
							FilterInfoPattern: configure.FilterInfoPattern{
								Processing: task.TypeProcessing,
								TaskIndex:  taskIndex,
								IPAddress:  task.RemoteIP,
							},
							FilterCountPattern: configure.FilterCountPattern{
								CountFilesFound:       task.CountFilesFound,
								CountCycleComplete:    task.CountCycleComplete,
								CountFoundFilesSize:   task.CountFoundFilesSize,
								CountFilesProcessed:   task.CountFilesProcessed,
								CountFilesUnprocessed: task.CountFilesUnprocessed,
							},
						},
					}

					formatJSON, err := json.Marshal(&messageTypeFilteringComplete)
					if err != nil {
						_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
					}

					if err := sourceData.SendWsMessage(1, formatJSON); err != nil {
						_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
					}

					delete(ift.TaskID, taskIndex)
				}
			}
		}
	}
}

func processMsgFilterComingChannel(acc *configure.AccessClientsConfigure, ift *configure.InformationFilteringTask) {
	sendStopOrCompleteMsg := func(taskIndex string, task *configure.TaskInformation, sourceData *configure.ClientsConfigure) {
		messageTypeFilteringComplete := configure.MessageTypeFilteringComplete{
			MessageType: "filtering",
			Info: configure.MessageTypeFilteringCompleteInfo{
				FilterInfoPattern: configure.FilterInfoPattern{
					Processing: task.TypeProcessing,
					TaskIndex:  taskIndex,
					IPAddress:  task.RemoteIP,
				},
				FilterCountPattern: configure.FilterCountPattern{
					CountFilesFound:       task.CountFilesFound,
					CountCycleComplete:    task.CountCycleComplete,
					CountFoundFilesSize:   task.CountFoundFilesSize,
					CountFilesProcessed:   task.CountFilesProcessed,
					CountFilesUnprocessed: task.CountFilesUnprocessed,
				},
			},
		}

		fmt.Println("CHANNEL, processing COMPLETE", taskIndex)

		formatJSON, err := json.Marshal(&messageTypeFilteringComplete)
		if err != nil {
			_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
		}

		if err := sourceData.SendWsMessage(1, formatJSON); err != nil {
			_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
		}

		delete(ift.TaskID, taskIndex)
	}

	for {
		msgInfoFilterTask := <-acc.ChanInfoFilterTask

		//fmt.Println("CHANNEL, received filtering task ID:", msgInfoFilterTask.TaskIndex)
		//fmt.Println(ift)

		if task, ok := ift.TaskID[msgInfoFilterTask.TaskIndex]; ok {
			task.RemoteIP = msgInfoFilterTask.RemoteIP
			task.TypeProcessing = msgInfoFilterTask.TypeProcessing

			//fmt.Println("CHANNEL, task ID is exist", msgInfoFilterTask.TaskIndex)
			_ = saveMessageApp.LogMessage("info", "CHANNEL, task ID is exist"+msgInfoFilterTask.TaskIndex)

			if sourceData, ok := acc.Addresses[task.RemoteIP]; ok {

				switch msgInfoFilterTask.TypeProcessing {
				case "execute":
					var mtfeou configure.MessageTypeFilteringExecutedOrUnexecuted
					mtfeou.MessageType = "filtering"
					mtfeou.Info.IPAddress = msgInfoFilterTask.RemoteIP
					mtfeou.Info.TaskIndex = msgInfoFilterTask.TaskIndex
					mtfeou.Info.Processing = msgInfoFilterTask.TypeProcessing
					mtfeou.Info.ProcessingFile.FileName = task.ProcessingFileName
					mtfeou.Info.ProcessingFile.DirectoryLocation = task.DirectoryFiltering
					mtfeou.Info.ProcessingFile.StatusProcessed = task.StatusProcessedFile
					mtfeou.Info.CountFilesProcessed = task.CountFilesProcessed
					mtfeou.Info.CountFilesUnprocessed = task.CountFilesUnprocessed
					mtfeou.Info.CountCycleComplete = task.CountCycleComplete
					mtfeou.Info.CountFilesFound = task.CountFilesFound
					mtfeou.Info.CountFoundFilesSize = task.CountFoundFilesSize

					//fmt.Println("CHANNEL, processing EXECUTE", msgInfoFilterTask.TaskIndex)

					formatJSON, err := json.Marshal(&mtfeou)
					if err != nil {
						_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
					}

					if err := sourceData.SendWsMessage(1, formatJSON); err != nil {
						_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
					}
				case "complete":
					sendStopOrCompleteMsg(msgInfoFilterTask.TaskIndex, task, sourceData)
				case "stop":
					sendStopOrCompleteMsg(msgInfoFilterTask.TaskIndex, task, sourceData)
				}
			}
		}
	}
}

//RouteWebSocketRequest маршрутизирует запросы
func RouteWebSocketRequest(remoteIP string, acc *configure.AccessClientsConfigure, ift *configure.InformationFilteringTask, mc *configure.MothConfig) {
	fmt.Println("*** RouteWebSocketRequest.go ***")

	c := acc.Addresses[remoteIP].WsConnection

	chanTypePing := make(chan []byte)
	chanTypeInfoFilterTask := make(chan configure.ChanInfoFilterTask, (acc.Addresses[remoteIP].MaxCountProcessFiltering * len(mc.CurrentDisks)))
	defer func() {
		close(chanTypePing)
		close(chanTypeInfoFilterTask)
	}()

	var prf configure.ParametrsFunctionRequestFilter

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

			go messageTypePing.RequestTypePing(remoteIP, mc.ExternalIPAddress, acc, chanTypePing)

			err = c.WriteMessage(1, <-chanTypePing)
			if err != nil {
				_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
			}

			//отправляем сообщение о выполняемой или выполненной задачи по фильтрации (если соединение было разорванно и вновь установленно)
			sendFilterTaskInfoAfterPingMessage(remoteIP, mc.ExternalIPAddress, acc, ift)

			//отправка системной информации подключенным источникам
			go func() {
				for {
					messageResponse := <-acc.ChanInfoTranssmition

					err = c.WriteMessage(1, messageResponse)
					if err != nil {
						_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))

						return
					}
				}
			}()

			//обработка информационных сообщений получаемых через канал ChanInfoFilterTask
			go processMsgFilterComingChannel(acc, ift)

		case "filtering":
			fmt.Println("routing to FILTERING...")

			if err = json.Unmarshal(message, &messageTypeFilter); err != nil {
				_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
			}

			prf.RemoteIP = remoteIP
			prf.ExternalIP = mc.ExternalIPAddress
			prf.CurrentDisks = mc.CurrentDisks
			prf.PathStorageFilterFiles = mc.PathStorageFilterFiles
			prf.TypeAreaNetwork = mc.TypeAreaNetwork
			prf.AccessClientsConfigure = acc

			go processingWebsocketRequest.RequestTypeFilter(&prf, &messageTypeFilter, ift)

		case "download files":
			fmt.Println("routing to DOWNLOAD FILES...")

		}
	}
}
