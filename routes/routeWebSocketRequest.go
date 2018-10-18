package routes

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"

	"moth_go/configure"
	"moth_go/helpers"
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
var messageTypeDownloadFiles configure.MessageTypeDownloadFiles

//sendCompleteMsg отправляет сообщение о завершении фильтрации
func sendCompleteMsg(acc *configure.AccessClientsConfigure, ift *configure.InformationFilteringTask, taskIndex string, task *configure.TaskInformation) {
	const sizeChunk = 30
	var messageTypeFilteringCompleteFirstPart configure.MessageTypeFilteringCompleteFirstPart

	//получить список найденных, в результате фильтрации, файлов
	getListFoundFiles := func(directoryResultFilter string) (list []configure.FoundFilesInfo, sizeFiles int64, err error) {
		files, err := ioutil.ReadDir(directoryResultFilter)
		if err != nil {
			return list, sizeFiles, err
		}

		var ffi configure.FoundFilesInfo
		for _, file := range files {
			if (file.Name() != "readme.txt") && (file.Size() > 24) {
				ffi.FileName = file.Name()
				ffi.FileSize = file.Size()

				sizeFiles += file.Size()

				list = append(list, ffi)
			}
		}

		return list, sizeFiles, nil
	}

	//получить количество частей сообщений
	getCountPartsMessage := func(list []configure.FoundFilesInfo, sizeChunk int) int {
		maxFiles := float64(len(list))

		newCountChunk := float64(sizeChunk)
		x := math.Floor(maxFiles / newCountChunk)
		y := maxFiles / newCountChunk

		if (y - x) != 0 {
			x++
		}

		return int(x)
	}

	listFoundFiles, sizeFiles, err := getListFoundFiles(task.DirectoryFiltering)
	if err != nil {
		_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
	}

	messageTypeFilteringCompleteFirstPart = configure.MessageTypeFilteringCompleteFirstPart{
		MessageType: "filtering",
		Info: configure.MessageTypeFilteringCompleteInfoFirstPart{
			FilterInfoPattern: configure.FilterInfoPattern{
				Processing: task.TypeProcessing,
				TaskIndex:  taskIndex,
				IPAddress:  task.RemoteIP,
			},
			FilterCountPattern: configure.FilterCountPattern{
				CountFilesFound:       len(listFoundFiles),
				CountFoundFilesSize:   sizeFiles,
				CountCycleComplete:    task.CountCycleComplete,
				CountFilesProcessed:   task.CountFilesProcessed,
				CountFilesUnprocessed: task.CountFilesUnprocessed,
			},
		},
	}

	if len(listFoundFiles) == 0 {

		fmt.Println("--------------------- FILTERING COMPLETE (FILES NOT FOUND) -------------------")
		fmt.Println(messageTypeFilteringCompleteFirstPart)

		fmt.Println("++++++ job status: ", task.TypeProcessing, ", task ID:", taskIndex, "count files found:", task.CountFilesFound)

		formatJSON, err := json.Marshal(&messageTypeFilteringCompleteFirstPart)
		if err != nil {
			_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
		}

		if _, ok := acc.Addresses[task.RemoteIP]; ok {
			acc.ChanWebsocketTranssmition <- formatJSON
		}
	} else {
		countPartsMessage := getCountPartsMessage(listFoundFiles, sizeChunk)

		numberMessageParts := [2]int{0, countPartsMessage}
		messageTypeFilteringCompleteFirstPart.Info.NumberMessageParts = numberMessageParts

		//отправляется первая часть сообщения
		formatJSON, err := json.Marshal(&messageTypeFilteringCompleteFirstPart)
		if err != nil {
			_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
		}

		if _, ok := acc.Addresses[task.RemoteIP]; ok {
			acc.ChanWebsocketTranssmition <- formatJSON
		}

		//шаблон последующих частей сообщения
		var messageTypeFilteringCompleteSecondPart configure.MessageTypeFilteringCompleteSecondPart
		messageTypeFilteringCompleteSecondPart = configure.MessageTypeFilteringCompleteSecondPart{
			MessageType: "filtering",
			Info: configure.MessageTypeFilteringCompleteInfoSecondPart{
				FilterInfoPattern: configure.FilterInfoPattern{
					Processing: task.TypeProcessing,
					TaskIndex:  taskIndex,
					IPAddress:  task.RemoteIP,
				},
			},
		}

		//отправляются последующие части сообщений содержащие списки имен файлов
		for i := 1; i <= countPartsMessage; i++ {
			//получаем срез с частью имен файлов
			listFilesFoundDuringFiltering := helpers.GetChunkListFilesFoundDuringFiltering(configure.ChunkListParameters{
				NumPart:        i,
				CountParts:     countPartsMessage,
				SizeChunk:      sizeChunk,
				ListFoundFiles: listFoundFiles,
			})

			numberMessageParts[0] = i
			messageTypeFilteringCompleteSecondPart.Info.NumberMessageParts = numberMessageParts
			messageTypeFilteringCompleteSecondPart.Info.ListFilesFoundDuringFiltering = listFilesFoundDuringFiltering

			formatJSON, err := json.Marshal(&messageTypeFilteringCompleteSecondPart)
			if err != nil {
				_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
			}

			if _, ok := acc.Addresses[task.RemoteIP]; ok {
				acc.ChanWebsocketTranssmition <- formatJSON
			}
		}
	}

	delete(ift.TaskID, taskIndex)
	_ = saveMessageApp.LogMessage("info", task.TypeProcessing+" of the filter task execution with ID"+taskIndex)
}

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
						_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
					}

					if _, ok := acc.Addresses[task.RemoteIP]; ok {
						acc.ChanWebsocketTranssmition <- formatJSON
					}
				case "complete":
					sendCompleteMsg(acc, ift, taskIndex, task)
				}
			}
		}
	}
}

//processMsgFilterComingChannel обрабатывает иформацию о фильтрации получаемую из канала
func processMsgFilterComingChannel(acc *configure.AccessClientsConfigure, ift *configure.InformationFilteringTask) {
	sendStopMsg := func(taskIndex string, task *configure.TaskInformation, sourceData *configure.ClientsConfigure) {
		MessageTypeFilteringStop := configure.MessageTypeFilteringStop{
			MessageType: "filtering",
			Info: configure.MessageTypeFilteringStopInfo{
				configure.FilterInfoPattern{
					Processing: task.TypeProcessing,
					TaskIndex:  taskIndex,
					IPAddress:  task.RemoteIP,
				},
			},
		}

		fmt.Println("--------------------- FILTERING COMPLETE -------------------")
		fmt.Println(MessageTypeFilteringStop)

		fmt.Println("++++++ job status: ", task.TypeProcessing, ", task ID:", taskIndex, "count files found:", task.CountFilesFound)

		formatJSON, err := json.Marshal(&MessageTypeFilteringStop)
		if err != nil {
			_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
		}

		if _, ok := acc.Addresses[task.RemoteIP]; ok {
			acc.ChanWebsocketTranssmition <- formatJSON
		}

		delete(ift.TaskID, taskIndex)
		_ = saveMessageApp.LogMessage("info", task.TypeProcessing+" of the filter task execution with ID"+taskIndex)
	}

	for {
		msgInfoFilterTask := <-acc.ChanInfoFilterTask

		if task, ok := ift.TaskID[msgInfoFilterTask.TaskIndex]; ok {

			//fmt.Println("====== RESIVED FROM CHAN MSG type processing", msgInfoFilterTask.TypeProcessing, "TYPE PROCESSING SAVE TASK", task.TypeProcessing)
			fmt.Println("====== RESIVED FROM CHAN", task.CountFilesProcessed)

			task.RemoteIP = msgInfoFilterTask.RemoteIP
			task.CountFilesFound = msgInfoFilterTask.CountFilesFound
			task.CountFoundFilesSize = msgInfoFilterTask.CountFoundFilesSize
			task.ProcessingFileName = msgInfoFilterTask.ProcessingFileName
			task.StatusProcessedFile = msgInfoFilterTask.StatusProcessedFile

			task.CountFilesProcessed++
			task.CountCycleComplete++

			if sourceData, ok := acc.Addresses[task.RemoteIP]; ok {

				switch msgInfoFilterTask.TypeProcessing {
				case "execute":
					if (task.TypeProcessing == "stop") || (task.TypeProcessing == "complete") {
						continue
					}
					fmt.Println("++++++ job status: ", task.TypeProcessing, ", task ID:", msgInfoFilterTask.TaskIndex, "count files found:", task.CountFilesFound)

					mtfeou := configure.MessageTypeFilteringExecutedOrUnexecuted{
						MessageType: "filtering",
						Info: configure.MessageTypeFilteringExecuteOrUnexecuteInfo{
							configure.FilterInfoPattern{
								IPAddress:  msgInfoFilterTask.RemoteIP,
								TaskIndex:  msgInfoFilterTask.TaskIndex,
								Processing: msgInfoFilterTask.TypeProcessing,
							},
							configure.FilterCountPattern{
								CountFilesProcessed:   task.CountFilesProcessed,
								CountFilesUnprocessed: task.CountFilesUnprocessed,
								CountCycleComplete:    task.CountCycleComplete,
								CountFilesFound:       msgInfoFilterTask.CountFilesFound,
								CountFoundFilesSize:   msgInfoFilterTask.CountFoundFilesSize,
							},
							configure.InfoProcessingFile{
								FileName:          msgInfoFilterTask.ProcessingFileName,
								DirectoryLocation: msgInfoFilterTask.DirectoryName,
								StatusProcessed:   msgInfoFilterTask.StatusProcessedFile,
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
					sendCompleteMsg(acc, ift, msgInfoFilterTask.TaskIndex, task)
				case "stop":
					sendStopMsg(msgInfoFilterTask.TaskIndex, task, sourceData)
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
			_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
		}

		if _, ok := acc.Addresses[remoteIP]; ok {
			acc.ChanWebsocketTranssmition <- formatJSON
		}
	}

	for {
		msgInfoDownloadTask := <-acc.ChanInfoDownloadTaskSendMoth

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
					_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
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
					_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
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
}

//RouteWebSocketRequest маршрутизирует запросы
func RouteWebSocketRequest(remoteIP string, acc *configure.AccessClientsConfigure, ift *configure.InformationFilteringTask, dfi *configure.DownloadFilesInformation, mc *configure.MothConfig) {
	c := acc.Addresses[remoteIP].WsConnection

	chanTypePing := make(chan []byte)
	chanTypeInfoFilterTask := make(chan configure.ChanInfoFilterTask, (acc.Addresses[remoteIP].MaxCountProcessFiltering * len(mc.CurrentDisks)))

	defer func() {
		close(chanTypePing)
		close(chanTypeInfoFilterTask)
	}()

	var prf configure.ParametrsFunctionRequestFilter
	var pfrdf configure.ParametrsFunctionRequestDownloadFiles

	for {
		_, message, err := c.ReadMessage()
		if err != nil {
			_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
			break
		}
		if err = json.Unmarshal(message, &messageType); err != nil {
			_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
		}

		fmt.Println("************* RESIVED MESSAGE", messageType.Type, "********* funcRouteWebsocketRequest *********")

		switch messageType.Type {
		case "ping":
			fmt.Println("routing to PING...")

			if err = json.Unmarshal(message, &messageTypePing); err != nil {
				_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
			}

			go messageTypePing.RequestTypePing(remoteIP, mc.ExternalIPAddress, acc, chanTypePing)

			messagePing := <-chanTypePing
			if _, ok := acc.Addresses[remoteIP]; ok {
				acc.ChanWebsocketTranssmition <- messagePing
			}

			//отправляем сообщение о выполняемой или выполненной задачи по фильтрации (выполняется при повторном установлении соединения)
			sendFilterTaskInfoAfterPingMessage(remoteIP, mc.ExternalIPAddress, acc, ift)

			//отправка системной информации подключенным источникам
			go func() {
				for {
					//fmt.Println("<--- TRANSSMITION SYSTEM INFORMATION!!!")

					messageResponse := <-acc.ChanInfoTranssmition

					if _, ok := acc.Addresses[remoteIP]; ok {
						acc.ChanWebsocketTranssmition <- messageResponse
					}
				}
			}()

			//обработка информационных сообщений о фильтрации (канал ChanInfoFilterTask)
			go processMsgFilterComingChannel(acc, ift)

			//обработка информационных сообщений о выгрузке файлов (канал ChanInfoDownloadTask)
			go processMsgDownloadComingChannel(acc, dfi, remoteIP)

		case "filtering":
			fmt.Println("*******-------- routing to FILTERING...")

			if err = json.Unmarshal(message, &messageTypeFilter); err != nil {
				_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
			}

			fmt.Println("-----------------------------", messageTypeFilter, "-----------------------------")

			prf.RemoteIP = remoteIP
			prf.ExternalIP = mc.ExternalIPAddress
			prf.CurrentDisks = mc.CurrentDisks
			prf.PathStorageFilterFiles = mc.PathStorageFilterFiles
			prf.TypeAreaNetwork = mc.TypeAreaNetwork
			prf.AccessClientsConfigure = acc

			processingWebsocketRequest.RequestTypeFilter(&prf, messageTypeFilter, ift)

		case "download files":
			fmt.Println("routing to DOWNLOAD FILES...")

			if err = json.Unmarshal(message, &messageTypeDownloadFiles); err != nil {
				_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
			}

			fmt.Println("-----------------------------", messageTypeDownloadFiles, "-----------------------------")

			pfrdf.RemoteIP = remoteIP
			pfrdf.ExternalIP = mc.ExternalIPAddress
			pfrdf.PathStorageFilterFiles = mc.PathStorageFilterFiles
			pfrdf.AccessClientsConfigure = acc

			processingWebsocketRequest.RequestTypeUploadFiles(&pfrdf, messageTypeDownloadFiles, dfi)
		}
	}
}
