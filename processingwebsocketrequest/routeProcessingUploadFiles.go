package processingwebsocketrequest

import (
	"moth_go/configure"
)

//RouteProcessingUploadFiles осуществляет обработку запросов на скачивание файлов
func RouteProcessingUploadFiles(pfrdf *configure.ParametrsFunctionRequestDownloadFiles, dfi *configure.DownloadFilesInformation, chanEndGorouting <-chan struct{}) {
	//канал для сообщений об успешной или не успешной передаче файла
	chanSendFile := make(chan configure.ChanSendFile)

	//канал информирующий о завершении передачи по причине исчерпания перечня выбранных для передачи файлов
	chanFinishDownloadFiles := make(chan struct{})

	defer func() {
		close(chanSendFile)
		close(chanFinishDownloadFiles)
	}()

	stopOrCancelTask := func(msgType string) {
		pfrdf.AccessClientsConfigure.ChanInfoDownloadTaskSendMoth <- configure.ChanInfoDownloadTask{
			TaskIndex:      dfi.RemoteIP[pfrdf.RemoteIP].TaskIndex,
			TypeProcessing: msgType,
			RemoteIP:       pfrdf.RemoteIP,
		}

		//удаляем задачу по скачиванию файлов
		dfi.DelTaskDownloadFiles(pfrdf.RemoteIP)
	}

DONE:
	for {
		select {
		case msgInfoDownloadTask := <-pfrdf.AccessClientsConfigure.ChanInfoDownloadTaskGetMoth:

			switch msgInfoDownloadTask.TypeProcessing {
			case "stop":
				dfi.ChangeStopTaskDownloadFiles(pfrdf.RemoteIP, dfi.RemoteIP[pfrdf.RemoteIP].TaskIndex, true)

			case "ready":
				//инициализация начала передачи файлов
				go ProcessingUploadFiles(chanFinishDownloadFiles, pfrdf, dfi, chanSendFile)

			case "waiting for transfer":
				//непосредственная передача файла
				go ReadSelectedFile(pfrdf, dfi)

			case "execute success":
				if ok := dfi.HasStopedTaskDownloadFiles(pfrdf.RemoteIP, dfi.RemoteIP[pfrdf.RemoteIP].TaskIndex); ok {
					stopOrCancelTask("stop")

					break DONE
				}

				chanSendFile <- "success"

			case "execute failure":
				if ok := dfi.HasStopedTaskDownloadFiles(pfrdf.RemoteIP, dfi.RemoteIP[pfrdf.RemoteIP].TaskIndex); ok {
					stopOrCancelTask("stop")

					break DONE
				}

				chanSendFile <- "failure"

			}

		case <-chanFinishDownloadFiles:
			stopOrCancelTask("completed")

			break DONE

		case <-chanEndGorouting:
			break DONE
		}
	}
}
