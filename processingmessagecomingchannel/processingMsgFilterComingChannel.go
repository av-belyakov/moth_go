package processingmessagecomingchannel

import (
	"encoding/json"
	"fmt"

	"moth_go/configure"
	"moth_go/processingwebsocketrequest"
	"moth_go/savemessageapp"
)

//ProcessMsgFilterComingChannel обрабатывает иформацию о фильтрации получаемую из канала
func ProcessMsgFilterComingChannel(acc *configure.AccessClientsConfigure, ift *configure.InformationFilteringTask) {
	//инициализируем функцию конструктор для записи лог-файлов
	saveMessageApp := savemessageapp.New()

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

	for msgInfoFilterTask := range acc.ChanInfoFilterTask {
		if task, ok := ift.TaskID[msgInfoFilterTask.TaskIndex]; ok {
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
					processingwebsocketrequest.SendMsgFilteringComplite(acc, ift, msgInfoFilterTask.TaskIndex, task)
				case "stop":
					sendStopMsg(msgInfoFilterTask.TaskIndex, task, sourceData)
				}
			}
		}
	}
}
