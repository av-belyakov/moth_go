package processingWebsocketRequest

import (
	"fmt"
	"moth_go/configure"
	"moth_go/errorMessage"
	"moth_go/helpers"
	"moth_go/saveMessageApp"
)

//отправляем сообщение о готовности к передаче файлов
func sendMessageReady(chanSendMoth chan<- configure.ChanInfoDownloadTask, taskIndex, remoteIP string) {
	chanSendMoth <- configure.ChanInfoDownloadTask{
		TaskIndex:      taskIndex,
		TypeProcessing: "ready",
		RemoteIP:       remoteIP,
	}
}

//RequestTypeDownloadFiles выполняет подготовку к выполнению задачи по выгрузки файлов
func RequestTypeUploadFiles(pfrdf *configure.ParametrsFunctionRequestDownloadFiles, mtdf configure.MessageTypeDownloadFiles, dfi *configure.DownloadFilesInformation) {
	fmt.Println("START function RequestTypeDownloadFiles...")

	if mtdf.Info.Processing == "start" {
		if (dfi.HasRemoteIPDownloadFiles(pfrdf.RemoteIP)) && (!dfi.HasTaskDownloadFiles(pfrdf.RemoteIP, mtdf.Info.TaskIndex)) {
			if err := errorMessage.SendErrorMessage(errorMessage.Options{
				RemoteIP:   pfrdf.RemoteIP,
				ErrMsg:     "limitTasks",
				TaskIndex:  mtdf.Info.TaskIndex,
				ExternalIP: pfrdf.ExternalIP,
				Wsc:        pfrdf.AccessClientsConfigure.Addresses[pfrdf.RemoteIP].WsConnection,
			}); err != nil {
				_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
			}

			return
		}

		//проверяем входные параметры (наличие директории с файлами если тип процесса start)
		if errMsg, ok := helpers.InputParametersForDownloadFile(pfrdf.RemoteIP, mtdf, dfi); !ok {
			fmt.Println("ERROR CHECK user input", errMsg)
			fmt.Println(ok)

			if err := errorMessage.SendErrorMessage(errorMessage.Options{
				RemoteIP:   pfrdf.RemoteIP,
				ErrMsg:     errMsg,
				TaskIndex:  mtdf.Info.TaskIndex,
				ExternalIP: pfrdf.ExternalIP,
				Wsc:        pfrdf.AccessClientsConfigure.Addresses[pfrdf.RemoteIP].WsConnection,
			}); err != nil {
				_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
			}
			return
		}

		//при загрузки всех файлов, без комбинирования списка
		if !mtdf.Info.DownloadSelectedFiles {
			fmt.Println("+++++ START DOWNLOAD FILES without combining the lists of files ++++++")

			go RouteProcessingUploadFiles(pfrdf, dfi)

			//отправляем сообщение о готовности к передаче файлов
			sendMessageReady(pfrdf.AccessClientsConfigure.ChanInfoDownloadTaskSendMoth, dfi.RemoteIP[pfrdf.RemoteIP].TaskIndex, pfrdf.RemoteIP)
		} else {
			//объединение списков файлов переданных клиентом если установлен флаг DownloadSelectedFiles = true
			layoutListCompleted, err := helpers.MergingFileListForTaskDownloadFiles(pfrdf, mtdf, dfi)
			if err != nil {
				if err := errorMessage.SendErrorMessage(errorMessage.Options{
					RemoteIP:   pfrdf.RemoteIP,
					ErrMsg:     "unexpectedValue",
					TaskIndex:  mtdf.Info.TaskIndex,
					ExternalIP: pfrdf.ExternalIP,
					Wsc:        pfrdf.AccessClientsConfigure.Addresses[pfrdf.RemoteIP].WsConnection,
				}); err != nil {
					_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
				}
				return
			}

			//если компоновка списка не завершена
			if !layoutListCompleted {
				fmt.Println("COMPARE MESSAGE LIST NOT COMPLETED")

				return
			}
			fmt.Println("-----+++++ START DOWNLOAD FILES COMPARENT MESSAGE DOWNLOAD FILES +++++------")

			go RouteProcessingUploadFiles(pfrdf, dfi)

			//отправляем сообщение о готовности к передаче файлов
			sendMessageReady(pfrdf.AccessClientsConfigure.ChanInfoDownloadTaskSendMoth, dfi.RemoteIP[pfrdf.RemoteIP].TaskIndex, pfrdf.RemoteIP)
		}

		return
	}

	//для всех типов событий кроме 'start'
	if dfi.HasTaskDownloadFiles(pfrdf.RemoteIP, mtdf.Info.TaskIndex) {
		pfrdf.AccessClientsConfigure.ChanInfoDownloadTaskGetMoth <- configure.ChanInfoDownloadTask{
			TaskIndex:      mtdf.Info.TaskIndex,
			TypeProcessing: mtdf.Info.Processing,
			RemoteIP:       pfrdf.RemoteIP,
		}
	} else {
		if err := errorMessage.SendErrorMessage(errorMessage.Options{
			RemoteIP:   pfrdf.RemoteIP,
			ErrMsg:     "noCoincidenceID",
			TaskIndex:  mtdf.Info.TaskIndex,
			ExternalIP: pfrdf.ExternalIP,
			Wsc:        pfrdf.AccessClientsConfigure.Addresses[pfrdf.RemoteIP].WsConnection,
		}); err != nil {
			_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
		}
	}
}