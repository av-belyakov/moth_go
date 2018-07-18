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
func RequestTypeDownloadFiles(pfrdf configure.ParametrsFunctionRequestDownloadFiles, mtdf configure.MessageTypeDownloadFiles, dfi *configure.DownloadFilesInformation) {
	fmt.Println("START function RequestTypeDownloadFiles...")

	fmt.Println("--------------- mtdf ---------------", mtdf)

	/*
					проверяем количество одновременно выполняемых задач
		запускать go подпрограмму только если с указанного ip адреса не было задач на выгрузку файлов
	*/
	if !dfi.HasTaskDownloadFiles(pfrdf.RemoteIP) {
		if mtdf.Info.Processing != "start" {
			if err := errorMessage.SendErrorMessage(errorMessage.Options{
				RemoteIP:   pfrdf.RemoteIP,
				ErrMsg:     "badRequest",
				TaskIndex:  mtdf.Info.TaskIndex,
				ExternalIP: pfrdf.ExternalIP,
				Wsc:        pfrdf.AccessClientsConfigure.Addresses[pfrdf.RemoteIP].WsConnection,
			}); err != nil {
				_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
			}
			return
		}

		//проверяем входные параметры (наличие директории с файлами)
		if errMsg, ok := helpers.InputParametersForDownloadFile(pfrdf.RemoteIP, mtdf, dfi); !ok {
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

		if !mtdf.Info.DownloadSelectedFiles {
			fmt.Println("+++++ START DOWNLOAD FILES without combining the lists of files ++++++")

			go ProcessingDownloadFiles(pfrdf, dfi)

			//отправляем сообщение о готовности к передаче файлов
			sendMessageReady(pfrdf.AccessClientsConfigure.ChanInfoDownloadTaskSendMoth, dfi.RemoteIP[pfrdf.RemoteIP].TaskIndex, pfrdf.RemoteIP)
		}

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

		go ProcessingDownloadFiles(pfrdf, dfi)

		//отправляем сообщение о готовности к передаче файлов
		sendMessageReady(pfrdf.AccessClientsConfigure.ChanInfoDownloadTaskSendMoth, dfi.RemoteIP[pfrdf.RemoteIP].TaskIndex, pfrdf.RemoteIP)
	} else {

	}
}
