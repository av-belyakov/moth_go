package processingWebsocketRequest

import (
	"fmt"
	"moth_go/configure"
	"moth_go/errorMessage"
	"moth_go/helpers"
	"moth_go/saveMessageApp"
)

//RequestTypeDownloadFiles выполняет подготовку к выполнению задачи по выгрузки файлов
func RequestTypeDownloadFiles(pfrdf configure.ParametrsFunctionRequestDownloadFiles, mtdf configure.MessageTypeDownloadFiles, dfi *configure.DownloadFilesInformation) {
	fmt.Println("START function RequestTypeDownloadFiles...")

	fmt.Println("--------------- mtdf ---------------", mtdf)

	switch mtdf.Info.Processing {
	case "start":

		//проверяем количество одновременно выполняемых задач
		if dfi.HasTaskDownloadFiles(pfrdf.RemoteIP) {
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

		if mtdf.Info.DownloadSelectedFiles {
			//объединение списков файлов переданных клиентом
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
				return
			}
			fmt.Println("-----+++++ COMPARENT MESSAGE DOWNLOAD FILES +++++------")

		}

		fmt.Println("------- TEST START DOWNLOAD FILES MESSAGE -----", mtdf.Info.TaskIndex)

	case "stop":

	case "ready":

	case "waiting for transfer":

	case "execute success":

	case "execute failure":

	}

}
