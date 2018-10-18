package processingWebsocketRequest

import (
	"fmt"
	"moth_go/configure"
)

//RouteProcessingUploadFiles осуществляет обработку запросов на скачивание файлов
func RouteProcessingUploadFiles(pfrdf *configure.ParametrsFunctionRequestDownloadFiles, dfi *configure.DownloadFilesInformation) {
	fmt.Println("*************** DOWNLOADING, function ProcessingDownloadFiles START...")
	fmt.Println("--- 2 /////////////////////// dfi listFiles ", dfi.RemoteIP[pfrdf.RemoteIP].ListDownloadFiles)

	//канал для сообщений об успешной или не успешной передаче файла
	chanSendFile := make(chan configure.ChanSendFile)

	//канал информирующий об остановки передачи файлов
	chanSendStopDownloadFiles := make(chan configure.ChanSendStopDownloadFiles)

	//канал для остановки чтения файла
	chanStopReadDownloadFiles := make(chan configure.ChanStopReadDownloadFile)

	labelStop := false

DONE:
	for {
		fmt.Println("****ROUTING**** func ProcessingDownloadFiles package routeWebSocketRequest")

		if labelStop {
			return
		}

		select {
		case msgInfoDownloadTask := <-pfrdf.AccessClientsConfigure.ChanInfoDownloadTaskGetMoth:

			switch msgInfoDownloadTask.TypeProcessing {
			case "stop":
				fmt.Println("RESIVED MSG TYPE 'stop'", "drop user type 'DownloadFilesInformation'")

				fmt.Println("!!!!!! ВЫХОД ИЗ GO-ПОДПРОГРАММЫ processingUploadFiles ----------------")

				chanStopReadDownloadFiles <- struct{}{}

				//закрываем канал chanSendFile для выхода из go-подпрограммы 'ProcessingUploadFiles'
				close(chanSendFile)

				//очищаем список файлов выбранных для передачи
				dfi.ClearListFiles(pfrdf.RemoteIP)

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
				}

				labelStop = true

				//выход из цикла, завершение go-подпрограммы
				break DONE

			case "ready":
				fmt.Println("RESIVED MSG TYPE 'ready'")

				//инициализация начала передачи файлов
				go ProcessingUploadFiles(pfrdf, dfi, chanSendFile, chanSendStopDownloadFiles)

			case "waiting for transfer":
				fmt.Println("RESIVED MSG TYPE 'waiting for transfer'")
				fmt.Printf("%v", msgInfoDownloadTask)

				//непосредственная передача файла
				go ReadSelectedFile(pfrdf, dfi, chanStopReadDownloadFiles)

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
		case <-chanSendStopDownloadFiles:
			labelStop = true

			break DONE

		}
	}

	fmt.Println("Останов процесса выгрузки файлов, функция 'routeProcessingUploadFiles' ----")
}
