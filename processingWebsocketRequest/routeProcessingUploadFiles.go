package processingWebsocketRequest

import (
	"fmt"
	"moth_go/configure"
)

//RouteProcessingUploadFiles осуществляет обработку запросов на скачивание файлов
func RouteProcessingUploadFiles(pfrdf *configure.ParametrsFunctionRequestDownloadFiles, dfi *configure.DownloadFilesInformation, chanEndGorouting <-chan struct{}) {
	fmt.Println("*************** DOWNLOADING, function ProcessingDownloadFiles START...")
	fmt.Println("--- 2 /////////////////////// dfi listFiles ", dfi.RemoteIP[pfrdf.RemoteIP].ListDownloadFiles)

	//канал для сообщений об успешной или не успешной передаче файла
	chanSendFile := make(chan configure.ChanSendFile)

	//канал информирующий об остановки передачи файлов
	chanSendStopDownloadFiles := make(chan configure.ChanSendStopDownloadFiles)

	defer func() {
		close(chanSendFile)
		close(chanSendStopDownloadFiles)
	}()

	stopOrCancelTask := func(msgType string) {
		fmt.Println("...START func stopOrCancelTask")

		pfrdf.AccessClientsConfigure.ChanInfoDownloadTaskSendMoth <- configure.ChanInfoDownloadTask{
			TaskIndex:      dfi.RemoteIP[pfrdf.RemoteIP].TaskIndex,
			TypeProcessing: msgType,
			RemoteIP:       pfrdf.RemoteIP,
		}

		//закрываем канал chanSendFile для выхода из go-подпрограммы 'ProcessingUploadFiles'
		//close(chanSendFile)

		//удаляем задачу по скачиванию файлов
		dfi.DelTaskDownloadFiles(pfrdf.RemoteIP)
	}

	//labelExit := false

DONE:
	for {
		/*if labelExit {

			fmt.Println("resived label 'labelExit' is", labelExit)

			return
		}*/

		fmt.Println("****ROUTING**** func ProcessingDownloadFiles package routeWebSocketRequest")

		select {
		case msgInfoDownloadTask := <-pfrdf.AccessClientsConfigure.ChanInfoDownloadTaskGetMoth:

			switch msgInfoDownloadTask.TypeProcessing {
			case "stop":

				fmt.Println("!!!!!! отправка в канал chanStopReadFile и ВЫХОД ИЗ GO-ПОДПРОГРАММЫ processingUploadFiles ----------------")
				fmt.Println("change label 'labelExit' on TRUE")

				dfi.ChangeStopTaskDownloadFiles(pfrdf.RemoteIP, dfi.RemoteIP[pfrdf.RemoteIP].TaskIndex, true)

				//labelExit = true
			case "ready":
				fmt.Println("RESIVED MSG TYPE 'ready'")

				//инициализация начала передачи файлов
				go ProcessingUploadFiles(pfrdf, dfi, chanSendFile, chanSendStopDownloadFiles)

			case "waiting for transfer":
				fmt.Println("RESIVED MSG TYPE 'waiting for transfer'")
				fmt.Printf("%v", msgInfoDownloadTask)

				//непосредственная передача файла
				go ReadSelectedFile(pfrdf, dfi)

			case "execute success":
				fmt.Println("***** RESIVED MSG TYPE 'execute success', file name", msgInfoDownloadTask.InfoFileDownloadTask.FileName, " =====")

				if ok := dfi.HasStopedTaskDownloadFiles(pfrdf.RemoteIP, dfi.RemoteIP[pfrdf.RemoteIP].TaskIndex); ok {

					fmt.Println("func HasStopedTaskDownloadFiles == TRUE")

					stopOrCancelTask("stop")

					break DONE
				}

				fmt.Println("MSG TYPE 'execute success' -> send chanel MSG 'success'")

				chanSendFile <- "success"

			case "execute failure":
				fmt.Println("***** RESIVED MSG TYPE 'execute failure' file name", msgInfoDownloadTask.InfoFileDownloadTask.FileName, "=====")

				if ok := dfi.HasStopedTaskDownloadFiles(pfrdf.RemoteIP, dfi.RemoteIP[pfrdf.RemoteIP].TaskIndex); ok {

					fmt.Println("func HasStopedTaskDownloadFiles == TRUE")

					stopOrCancelTask("stop")

					break DONE
				}

				chanSendFile <- "failure"

			}

		case <-chanSendStopDownloadFiles:

			fmt.Println("---------- resived MSG type 'C_O_M_P_L_E_T_E_D' (func routeProcessingUploadFiles)")

			stopOrCancelTask("completed")

			break DONE

		case <-chanEndGorouting:
			break DONE
		}
	}

	fmt.Println("Останов процесса выгрузки файлов, функция 'routeProcessingUploadFiles' ----")
}
