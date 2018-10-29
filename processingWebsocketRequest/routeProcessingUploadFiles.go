package processingWebsocketRequest

import (
	"fmt"
	"moth_go/configure"
)

//RouteProcessingUploadFiles осуществляет обработку запросов на скачивание файлов
func RouteProcessingUploadFiles(pfrdf *configure.ParametrsFunctionRequestDownloadFiles, dfi *configure.DownloadFilesInformation) {
	fmt.Println("*************** DOWNLOADING, function ProcessingDownloadFiles START...")
	fmt.Println("--- 2 /////////////////////// dfi listFiles ", dfi.RemoteIP[pfrdf.RemoteIP].ListDownloadFiles)

	type labelStopInfo struct {
		isStoped bool
		msgType  string
	}

	//канал для сообщений об успешной или не успешной передаче файла
	chanSendFile := make(chan configure.ChanSendFile)

	//канал информирующий об остановки передачи файлов
	chanSendStopDownloadFiles := make(chan configure.ChanSendStopDownloadFiles)

	//канал для инициализации остановки чтения файла
	chanStopReadFile := make(chan configure.ChanStopReadFile)

	//канал информирующий о успешной остановке чтения файла
	chanStopedReadFile := make(chan configure.ChanStopedReadFile)

	lsi := labelStopInfo{}

DONE:
	for {
		fmt.Println("****ROUTING**** func ProcessingDownloadFiles package routeWebSocketRequest")

		if lsi.isStoped {

			fmt.Println(lsi, "<--->LABEL")

			pfrdf.AccessClientsConfigure.ChanInfoDownloadTaskSendMoth <- configure.ChanInfoDownloadTask{
				TaskIndex:      dfi.RemoteIP[pfrdf.RemoteIP].TaskIndex,
				TypeProcessing: lsi.msgType,
				RemoteIP:       pfrdf.RemoteIP,
			}

			//закрываем канал chanSendFile для выхода из go-подпрограммы 'ProcessingUploadFiles'
			close(chanSendFile)

			//удаляем задачу по скачиванию файлов
			dfi.DelTaskDownloadFiles(pfrdf.RemoteIP)

			return
		}

		select {
		case msgInfoDownloadTask := <-pfrdf.AccessClientsConfigure.ChanInfoDownloadTaskGetMoth:

			switch msgInfoDownloadTask.TypeProcessing {
			case "stop":

				fmt.Println("!!!!!! отправка в канал chanStopReadFile и ВЫХОД ИЗ GO-ПОДПРОГРАММЫ processingUploadFiles ----------------")

				//отправляем для  того что бы остановить чтение файла
				chanStopReadFile <- struct{}{}

			case "ready":
				fmt.Println("RESIVED MSG TYPE 'ready'")

				//инициализация начала передачи файлов
				go ProcessingUploadFiles(pfrdf, dfi, chanSendFile, chanSendStopDownloadFiles)

			case "waiting for transfer":
				fmt.Println("RESIVED MSG TYPE 'waiting for transfer'")
				fmt.Printf("%v", msgInfoDownloadTask)

				//непосредственная передача файла
				go ReadSelectedFile(chanStopedReadFile, pfrdf, dfi, chanStopReadFile)

			case "execute success":
				fmt.Println("***** RESIVED MSG TYPE 'execute success', file name", msgInfoDownloadTask.InfoFileDownloadTask.FileName, " =====")

				/*

					МОЖЕТ БЫТЬ СДЕЛАТЬ ПРОВЕРКУ ПО ИМЕНИ ФАЙЛА????

				*/

				chanSendFile <- "success"

			case "execute failure":
				fmt.Println("***** RESIVED MSG TYPE 'execute failure' file name", msgInfoDownloadTask.InfoFileDownloadTask.FileName, "=====")

				chanSendFile <- "failure"

			}

		case <-chanStopedReadFile:

			fmt.Println("---------- resived MSG type 'S_T_O_P' (func routeProcessingUploadFiles)")

			//очищаем список файлов выбранных для передачи
			/*			dfi.ClearListFiles(pfrdf.RemoteIP)

						fmt.Println("ListDownloadFiles equal 0?", len(dfi.RemoteIP[pfrdf.RemoteIP].ListDownloadFiles))

						//проверяем наличие файлов для передачи
						if len(dfi.RemoteIP[pfrdf.RemoteIP].ListDownloadFiles) == 0 {
							pfrdf.AccessClientsConfigure.ChanInfoDownloadTaskSendMoth <- configure.ChanInfoDownloadTask{
								TaskIndex:      dfi.RemoteIP[pfrdf.RemoteIP].TaskIndex,
								TypeProcessing: "stop",
								RemoteIP:       pfrdf.RemoteIP,
							}

							//удаляем задачу по скачиванию файлов
							dfi.DelTaskDownloadFiles(pfrdf.RemoteIP)
						}*/

			lsi.isStoped = true
			lsi.msgType = "stop"

			//выход из цикла, завершение go-подпрограммы
			break DONE

		case <-chanSendStopDownloadFiles:

			fmt.Println("---------- resived MSG type 'C_O_M_P_L_E_T_E_D' (func routeProcessingUploadFiles)")

			lsi.isStoped = true
			lsi.msgType = "completed"

			//выход из цикла, завершение go-подпрограммы
			break DONE

		}
	}

	fmt.Println("Останов процесса выгрузки файлов, функция 'routeProcessingUploadFiles' ----")
}
