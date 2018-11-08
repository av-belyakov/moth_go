package processingwebsocketrequest

import (
	"fmt"

	"moth_go/configure"
	"moth_go/errormessage"
	"moth_go/helpers"
	"moth_go/savemessageapp"
)

//RequestTypeFilter выполняет подготовку к обработки запросов по фильтрации
func RequestTypeFilter(prf *configure.ParametrsFunctionRequestFilter, mtf configure.MessageTypeFilter, ift *configure.InformationFilteringTask) {
	fmt.Println("START function RequestTypeFilter...")

	//проверяем количество одновременно выполняемых задач
	if ift.IsMaxConcurrentProcessFiltering(prf.RemoteIP, prf.AccessClientsConfigure.Addresses[prf.RemoteIP].MaxCountProcessFiltering) {
		if err := errormessage.SendErrorMessage(errormessage.Options{
			RemoteIP:   prf.RemoteIP,
			ErrMsg:     "limitTasks",
			TaskIndex:  mtf.Info.TaskIndex,
			ExternalIP: prf.ExternalIP,
			Wsc:        prf.AccessClientsConfigure.Addresses[prf.RemoteIP].WsConnection,
		}); err != nil {
			_ = savemessageapp.LogMessage("error", fmt.Sprint(err))
		}
		return
	}

	//проверка полученных от пользователя данных (дата и время, список адресов и сетей)
	if errMsg, ok := helpers.InputParametrsForFiltering(ift, &mtf); !ok {
		if err := errormessage.SendErrorMessage(errormessage.Options{
			RemoteIP:   prf.RemoteIP,
			ErrMsg:     errMsg,
			TaskIndex:  mtf.Info.TaskIndex,
			ExternalIP: prf.ExternalIP,
			Wsc:        prf.AccessClientsConfigure.Addresses[prf.RemoteIP].WsConnection,
		}); err != nil {
			_ = savemessageapp.LogMessage("error", fmt.Sprint(err))
		}
		return
	}

	if mtf.Info.Settings.UseIndexes {
		//объединение списков файлов для задачи (возобновляемой или выполняемой на основе индексов)
		layoutListCompleted, err := helpers.MergingFileListForTaskFilter(ift, &mtf)
		if err != nil {
			if err := errormessage.SendErrorMessage(errormessage.Options{
				RemoteIP:   prf.RemoteIP,
				ErrMsg:     "unexpectedValue",
				TaskIndex:  mtf.Info.TaskIndex,
				ExternalIP: prf.ExternalIP,
				Wsc:        prf.AccessClientsConfigure.Addresses[prf.RemoteIP].WsConnection,
			}); err != nil {
				_ = savemessageapp.LogMessage("error", fmt.Sprint(err))
			}
			return
		}

		fmt.Println("layoutListCompleted = ", layoutListCompleted)

		//если компоновка списка не завершена
		if !layoutListCompleted {
			return
		}

		fmt.Println("START FILTERING FOR INDEXES *****************")
	}

	fmt.Println("------- TEST START FILTERING MESSAGE -----", mtf.Info.TaskIndex)

	go ProcessingFiltering(prf, &mtf, ift)
}
