package routes

import (
	"encoding/json"
	"fmt"
	"runtime"

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
	for taskIndex, task := range ift.TaskID {
		fmt.Println("AFTER start CONNECT FROM TASKS, received filtering task ID:", taskIndex)

		if task.RemoteIP == remoteIP {
			fmt.Println("FROM TASKS", task.RemoteIP, " == ", remoteIP)

			if _, ok := acc.Addresses[task.RemoteIP]; ok {

				fmt.Println("TYPE MSG", task.TypeProcessing)

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

					fmt.Println("+++ function sendFilterTaskInfoAfterPingMessage, execute")
					fmt.Println(mtfeou)

					formatJSON, err := json.Marshal(&mtfeou)
					if err != nil {
						_ = savemessageapp.LogMessage("error", fmt.Sprint(err))
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
			_ = savemessageapp.LogMessage("error", fmt.Sprint(err))
		}

		if _, ok := acc.Addresses[remoteIP]; ok {
			acc.ChanWebsocketTranssmition <- formatJSON
		}
	}

	for msgInfoDownloadTask := range acc.ChanInfoDownloadTaskSendMoth {

		fmt.Println("++++++++++ START function processMsgDownloadComingChannel package routeWebSocketRequest ++++++++++")
		fmt.Println("-----------------------------------")
		fmt.Println("CHANNEL acc.ChanInfoDownloadTask, message...", msgInfoDownloadTask)
		fmt.Println("-----------------------------------")

		if _, ok := acc.Addresses[remoteIP]; ok {
			switch msgInfoDownloadTask.TypeProcessing {
			case "ready":

				fmt.Println("->->-> send MSG 'ready' to Flashlight ->->->")

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

				fmt.Println("->->-> send MSG 'execute' to Flashlight ->->->")
				fmt.Printf("%v", mtdfe)

				formatJSON, err := json.Marshal(&mtdfe)
				if err != nil {
					_ = savemessageapp.LogMessage("error", fmt.Sprint(err))
				}

				if _, ok := acc.Addresses[remoteIP]; ok {
					acc.ChanWebsocketTranssmition <- formatJSON
				}

			case "execute completed":
				fmt.Println("!!!---- file UPLOADING complite ----")

				mtdfe := configure.MessageTypeDownloadFilesExecute{
					MessageType: "download files",
					Info: configure.MessageTypeDownloadFilesInfoExecute{
						FileName: msgInfoDownloadTask.InfoFileDownloadTask.FileName,
						FileSize: msgInfoDownloadTask.InfoFileDownloadTask.FileSize,
					},
				}

				mtdfe.Info.Processing = msgInfoDownloadTask.TypeProcessing
				mtdfe.Info.TaskIndex = msgInfoDownloadTask.TaskIndex

				fmt.Println("->->-> send MSG 'execute completed' to Flashlight ->->->")
				fmt.Printf("%v", mtdfe)

				formatJSON, err := json.Marshal(&mtdfe)
				if err != nil {
					_ = savemessageapp.LogMessage("error", fmt.Sprint(err))
				}

				if _, ok := acc.Addresses[remoteIP]; ok {
					acc.ChanWebsocketTranssmition <- formatJSON
				}

			case "completed":

				fmt.Println("!!!---- file UPLOADING completed ----")

				sendMessageTypeReadyOrCompleted(msgInfoDownloadTask)

			case "stop":

				fmt.Println("!!!---- file UPLOADING stop ----")

				sendMessageTypeReadyOrCompleted(msgInfoDownloadTask)
			}
		}
	}

	fmt.Println("**** STOP GOROUTIN ----'processMsgDownloadComingChannel'-----")

}

//RouteWebSocketRequest маршрутизирует запросы
func RouteWebSocketRequest(remoteIP string, acc *configure.AccessClientsConfigure, ift *configure.InformationFilteringTask, dfi *configure.DownloadFilesInformation, mc *configure.MothConfig, cssit <-chan struct{}) {
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
			_ = savemessageapp.LogMessage("error", fmt.Sprint(err))
			break
		}
		if err = json.Unmarshal(message, &messageType); err != nil {
			_ = savemessageapp.LogMessage("error", fmt.Sprint(err))
		}

		fmt.Println("************* RESIVED MESSAGE", messageType.Type, "********* funcRouteWebsocketRequest *********")

		switch messageType.Type {
		case "ping":
			fmt.Println("routing to PING...")

			if err = json.Unmarshal(message, &messageTypePing); err != nil {
				_ = savemessageapp.LogMessage("error", fmt.Sprint(err))
			}

			msgTypePong, err := messageTypePing.RequestTypePing(remoteIP, mc.ExternalIPAddress, acc)
			if err != nil {
				_ = savemessageapp.LogMessage("error", fmt.Sprint(err))
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

						fmt.Println("<--- TRANSSMITION SYSTEM INFORMATION!!!")
						fmt.Println("_!!! CURRENT COUNT GOROUTINE:", runtime.NumGoroutine())

						if _, ok := acc.Addresses[remoteIP]; ok {
							acc.ChanWebsocketTranssmition <- messageResponse
						}

					case <-cssit:

						//при разрыве соединения останавливаем подпрограмму requestTypeUploadFiles
						chanEndGoroutinTypeUploadFile <- struct{}{}

						break DONE
					}
				}

				fmt.Println("...___ STOP GOROUTIN anonimus function type PING")
			}()

			//обработка информационных сообщений о фильтрации (канал ChanInfoFilterTask)
			//			go processMsgFilterComingChannel(acc, ift)

			//обработка информационных сообщений о выгрузке файлов (канал ChanInfoDownloadTask)
			go processMsgDownloadComingChannel(acc, dfi, remoteIP)

		case "filtering":
			fmt.Println("*******-------- routing to FILTERING...")

			if err = json.Unmarshal(message, &messageTypeFilter); err != nil {
				_ = savemessageapp.LogMessage("error", fmt.Sprint(err))
			}

			fmt.Println("-----------------------------", messageTypeFilter, "-----------------------------")

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
				_ = savemessageapp.LogMessage("error", fmt.Sprint(err))
			}

			fmt.Println("--routing to DOWNLOAD FILES...---", messageTypeDownloadFiles, "-----------------------------")

			fmt.Println("____----- COUNT GOROUTINE =", runtime.NumGoroutine())

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
