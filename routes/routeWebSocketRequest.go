package routes

import (
	"encoding/json"
	"fmt"
	"log"

	"moth_go/configure"
	"moth_go/processingwebsocketrequest"
	"moth_go/savemessageapp"
)

//MessageType содержит тип JSON сообщения
type MessageType struct {
	Type string `json:"messageType"`
}

var messageType MessageType
var messageTypePing processingwebsocketrequest.MessageTypePing
var messageTypeFilter configure.MessageTypeFilter
var messageTypeDownloadFiles configure.MessageTypeDownloadFiles

//sendFilterTaskInfo отправляет сообщение о выполняемой или выполненной задаче по фильтрации сет. трафика
func sendFilterTaskInfoAfterPingMessage(remoteIP, ExternalIP string, acc *configure.AccessClientsConfigure, ift *configure.InformationFilteringTask) {
	//инициализируем функцию конструктор для записи лог-файлов
	saveMessageApp := savemessageapp.New()

	for taskIndex, task := range ift.TaskID {
		if task.RemoteIP == remoteIP {
			if _, ok := acc.Addresses[task.RemoteIP]; ok {
				switch task.TypeProcessing {
				case "execute":
					mtfeou := configure.MessageTypeFilteringExecutedOrUnexecuted{
						MessageType: "filtering",
						Info: configure.MessageTypeFilteringExecuteOrUnexecuteInfo{
							configure.FilterInfoPattern{
								IPAddress:  task.RemoteIP,
								TaskIndex:  taskIndex,
								Processing: task.TypeProcessing,
							},
							configure.FilterCountPattern{
								CountFilesProcessed:   task.CountFilesProcessed,
								CountFilesUnprocessed: task.CountFilesUnprocessed,
								CountCycleComplete:    task.CountCycleComplete,
								CountFilesFound:       task.CountFilesFound,
								CountFoundFilesSize:   task.CountFoundFilesSize,
							},
							configure.InfoProcessingFile{
								FileName:          task.ProcessingFileName,
								DirectoryLocation: task.DirectoryFiltering,
								StatusProcessed:   task.StatusProcessedFile,
							},
						},
					}

					formatJSON, err := json.Marshal(&mtfeou)
					if err != nil {
						_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
					}

					if _, ok := acc.Addresses[task.RemoteIP]; ok {
						acc.ChanWebsocketTranssmition <- formatJSON
					}
				case "complete":
					processingwebsocketrequest.SendMsgFilteringComplite(acc, ift, taskIndex, task)
				}
			}
		}
	}
}

//processMsgDownloadComingChannel обработка информации по выгружаемым файлам, полученной через канал ChanInfoDownloadTask
func processMsgDownloadComingChannel(acc *configure.AccessClientsConfigure, dfi *configure.DownloadFilesInformation, remoteIP string) {
	//инициализируем функцию конструктор для записи лог-файлов
	saveMessageApp := savemessageapp.New()

	sendMessageTypeReadyOrCompleted := func(message configure.ChanInfoDownloadTask) {
		mtdfrf := configure.MessageTypeDownloadFilesReadyOrCompleted{
			MessageType: "download files",
			Info: configure.MessageTypeDownloadFilesInfoReadyOrCompleted{
				Processing: message.TypeProcessing,
				TaskIndex:  message.TaskIndex,
			},
		}

		formatJSON, err := json.Marshal(&mtdfrf)
		if err != nil {
			_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
		}

		if _, ok := acc.Addresses[remoteIP]; ok {
			acc.ChanWebsocketTranssmition <- formatJSON
		}
	}

	for msgInfoDownloadTask := range acc.ChanInfoDownloadTaskSendMoth {
		if _, ok := acc.Addresses[remoteIP]; ok {
			switch msgInfoDownloadTask.TypeProcessing {
			case "ready":
				sendMessageTypeReadyOrCompleted(msgInfoDownloadTask)

			case "execute":
				mtdfe := configure.MessageTypeDownloadFilesExecute{
					MessageType: "download files",
					Info: configure.MessageTypeDownloadFilesInfoExecute{
						FileName: msgInfoDownloadTask.InfoFileDownloadTask.FileName,
						FileHash: msgInfoDownloadTask.InfoFileDownloadTask.FileHash,
						FileSize: msgInfoDownloadTask.InfoFileDownloadTask.FileSize,
					},
				}

				mtdfe.Info.Processing = msgInfoDownloadTask.TypeProcessing
				mtdfe.Info.TaskIndex = msgInfoDownloadTask.TaskIndex

				formatJSON, err := json.Marshal(&mtdfe)
				if err != nil {
					_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
				}

				if _, ok := acc.Addresses[remoteIP]; ok {
					acc.ChanWebsocketTranssmition <- formatJSON
				}

			case "execute completed":
				mtdfe := configure.MessageTypeDownloadFilesExecute{
					MessageType: "download files",
					Info: configure.MessageTypeDownloadFilesInfoExecute{
						FileName: msgInfoDownloadTask.InfoFileDownloadTask.FileName,
						FileSize: msgInfoDownloadTask.InfoFileDownloadTask.FileSize,
					},
				}

				mtdfe.Info.Processing = msgInfoDownloadTask.TypeProcessing
				mtdfe.Info.TaskIndex = msgInfoDownloadTask.TaskIndex

				formatJSON, err := json.Marshal(&mtdfe)
				if err != nil {
					_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
				}

				if _, ok := acc.Addresses[remoteIP]; ok {
					acc.ChanWebsocketTranssmition <- formatJSON
				}

			case "completed":
				sendMessageTypeReadyOrCompleted(msgInfoDownloadTask)

			case "stop":
				sendMessageTypeReadyOrCompleted(msgInfoDownloadTask)

			}
		}
	}
}

//RouteWebSocketRequest маршрутизирует запросы
func RouteWebSocketRequest(remoteIP string, acc *configure.AccessClientsConfigure, ift *configure.InformationFilteringTask, dfi *configure.DownloadFilesInformation, mc *configure.MothConfig, cssit <-chan struct{}) {
	//инициализируем функцию конструктор для записи лог-файлов
	saveMessageApp := savemessageapp.New()

	c := acc.Addresses[remoteIP].WsConnection

	chanTypePing := make(chan []byte)
	chanTypeInfoFilterTask := make(chan configure.ChanInfoFilterTask, (acc.Addresses[remoteIP].MaxCountProcessFiltering * len(mc.CurrentDisks)))

	//для останова подпрограммы RequestTypeUploadFiles
	chanEndGoroutinTypeUploadFile := make(chan struct{})

	defer func() {
		close(chanTypePing)
		close(chanTypeInfoFilterTask)
	}()

	for {
		_, message, err := c.ReadMessage()
		if err != nil {
			c.Close()

			//удаляем информацию о соединении из типа acc
			delete(acc.Addresses, remoteIP)
			_ = saveMessageApp.LogMessage("info", "disconnect for IP address "+remoteIP)

			//при разрыве соединения удаляем задачу по скачиванию файлов
			dfi.DelTaskDownloadFiles(remoteIP)

			log.Println("websocket disconnect whis ip", remoteIP)

			_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
			break
		}
		if err = json.Unmarshal(message, &messageType); err != nil {
			_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
		}

		switch messageType.Type {
		case "ping":
			if err = json.Unmarshal(message, &messageTypePing); err != nil {
				_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
			}

			msgTypePong, err := messageTypePing.RequestTypePing(remoteIP, mc, acc)
			if err != nil {
				_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
			}

			if _, ok := acc.Addresses[remoteIP]; ok {
				acc.ChanWebsocketTranssmition <- msgTypePong
			}

			//отправляем сообщение о выполняемой или выполненной задачи по фильтрации (выполняется при повторном установлении соединения)
			sendFilterTaskInfoAfterPingMessage(remoteIP, mc.ExternalIPAddress, acc, ift)

			//отправка системной информации подключенным источникам
			go func() {
			DONE:
				for {
					select {
					case messageResponse := <-acc.ChanInfoTranssmition:
						if _, ok := acc.Addresses[remoteIP]; ok {
							acc.ChanWebsocketTranssmition <- messageResponse
						}

					case <-cssit:
						//при разрыве соединения останавливаем подпрограмму requestTypeUploadFiles
						chanEndGoroutinTypeUploadFile <- struct{}{}

						break DONE
					}
				}
			}()

			//обработка информационных сообщений о фильтрации (канал ChanInfoFilterTask)
			//			go processMsgFilterComingChannel(acc, ift)

			//обработка информационных сообщений о выгрузке файлов (канал ChanInfoDownloadTask)
			go processMsgDownloadComingChannel(acc, dfi, remoteIP)

		case "filtering":
			if err = json.Unmarshal(message, &messageTypeFilter); err != nil {
				_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
			}

			_ = saveMessageApp.LogMessage("info", "resived message FILTERING type")

			prf := configure.ParametrsFunctionRequestFilter{
				RemoteIP:               remoteIP,
				ExternalIP:             mc.ExternalIPAddress,
				CurrentDisks:           mc.CurrentDisks,
				PathStorageFilterFiles: mc.PathStorageFilterFiles,
				TypeAreaNetwork:        mc.TypeAreaNetwork,
				AccessClientsConfigure: acc,
			}

			processingwebsocketrequest.RequestTypeFilter(&prf, messageTypeFilter, ift)

		case "download files":
			if err = json.Unmarshal(message, &messageTypeDownloadFiles); err != nil {
				_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
			}

			pfrdf := configure.ParametrsFunctionRequestDownloadFiles{
				RemoteIP:               remoteIP,
				ExternalIP:             mc.ExternalIPAddress,
				PathStorageFilterFiles: mc.PathStorageFilterFiles,
				AccessClientsConfigure: acc,
			}

			processingwebsocketrequest.RequestTypeUploadFiles(&pfrdf, messageTypeDownloadFiles, dfi, chanEndGoroutinTypeUploadFile)
		}
	}
}
