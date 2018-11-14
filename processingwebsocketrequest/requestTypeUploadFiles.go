package processingwebsocketrequest

import (
	"fmt"

	"moth_go/configure"
	"moth_go/errormessage"
	"moth_go/helpers"
	"moth_go/savemessageapp"
)

//отправляем сообщение о готовности к передаче файлов
func sendMessageReady(chanSendMoth chan<- configure.ChanInfoDownloadTask, taskIndex, remoteIP string) {
	chanSendMoth <- configure.ChanInfoDownloadTask{
		TaskIndex:      taskIndex,
		TypeProcessing: "ready",
		RemoteIP:       remoteIP,
	}
}

//RequestTypeUploadFiles выполняет подготовку к выполнению задачи по выгрузки файлов
func RequestTypeUploadFiles(pfrdf *configure.ParametrsFunctionRequestDownloadFiles, mtdf configure.MessageTypeDownloadFiles, dfi *configure.DownloadFilesInformation, chanEndGorouting <-chan struct{}) {
	//инициализируем функцию конструктор для записи лог-файлов
	saveMessageApp := savemessageapp.New()

	if mtdf.Info.Processing == "start" {
		if (dfi.HasRemoteIPDownloadFiles(pfrdf.RemoteIP)) && (!dfi.HasTaskDownloadFiles(pfrdf.RemoteIP, mtdf.Info.TaskIndex)) {
			if err := errormessage.SendErrorMessage(errormessage.Options{
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
			if err := errormessage.SendErrorMessage(errormessage.Options{
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
			go RouteProcessingUploadFiles(pfrdf, dfi, chanEndGorouting)

		} else {
			//объединение списков файлов переданных клиентом если установлен флаг DownloadSelectedFiles = true
			layoutListCompleted, err := helpers.MergingFileListForTaskDownloadFiles(pfrdf, mtdf, dfi)
			if err != nil {
				if err := errormessage.SendErrorMessage(errormessage.Options{
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
				return
			}

			go RouteProcessingUploadFiles(pfrdf, dfi, chanEndGorouting)

		}

		//отправляем сообщение о готовности к передаче файлов
		sendMessageReady(pfrdf.AccessClientsConfigure.ChanInfoDownloadTaskSendMoth, dfi.RemoteIP[pfrdf.RemoteIP].TaskIndex, pfrdf.RemoteIP)

		return
	}

	//для всех типов событий кроме 'start'
	if dfi.HasTaskDownloadFiles(pfrdf.RemoteIP, mtdf.Info.TaskIndex) {
		idt := configure.ChanInfoDownloadTask{
			TaskIndex:      mtdf.Info.TaskIndex,
			TypeProcessing: mtdf.Info.Processing,
			RemoteIP:       pfrdf.RemoteIP,
		}

		if mtdf.Info.Processing == "ready" || mtdf.Info.Processing == "stop" {
			pfrdf.AccessClientsConfigure.ChanInfoDownloadTaskGetMoth <- idt

			return
		}

		idt.InfoFileDownloadTask = configure.InfoFileDownloadTask{
			FileName: mtdf.Info.FileInformation.FileName,
			FileHash: mtdf.Info.FileInformation.FileHash,
		}

		pfrdf.AccessClientsConfigure.ChanInfoDownloadTaskGetMoth <- idt

	} else {
		if mtdf.Info.Processing == "execute success" || mtdf.Info.Processing == "execute failure" {
			return
		}

		if err := errormessage.SendErrorMessage(errormessage.Options{
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
