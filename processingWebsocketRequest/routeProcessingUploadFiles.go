package processingWebsocketRequest

import (
	"fmt"
	"moth_go/configure"
)

//RouteProcessingUploadFiles осуществляет обработку запросов на скачивание файлов
func RouteProcessingUploadFiles(pfrdf *configure.ParametrsFunctionRequestDownloadFiles, dfi *configure.DownloadFilesInformation) {
	fmt.Println("*************** DOWNLOADING, function ProcessingDownloadFiles START...")
	fmt.Println("--- 2 /////////////////////// dfi listFiles ", dfi.RemoteIP[pfrdf.RemoteIP].ListDownloadFiles)

	//канал с информацией об успешной или не успешной передаче файла
	chanSendFile := make(chan configure.ChanSendFile)

	for {
		msgInfoDownloadTask := <-pfrdf.AccessClientsConfigure.ChanInfoDownloadTaskGetMoth

		fmt.Println("****ROUTING**** func ProcessingDownloadFiles package routeWebSocketRequest")

		switch msgInfoDownloadTask.TypeProcessing {
		/*		case "start":
				fmt.Println("RESIVED MSG TYPE 'start'")
		*/
		case "stop":
			fmt.Println("RESIVED MSG TYPE 'stop'", "drop user type 'DownloadFilesInformation'")

		case "cancel":
			fmt.Println("RESIVED MSG TYPE 'cancel'")
			//необходимо остановить выгрузку файлов и удалить запись о задаче
			dfi.DelTaskDownloadFiles(pfrdf.RemoteIP)

			fmt.Println(dfi)

		case "ready":
			fmt.Println("RESIVED MSG TYPE 'ready'")

			//инициализация начала передачи файлов
			go ProcessingUploadFiles(pfrdf, dfi, chanSendFile)

		case "waiting for transfer":
			fmt.Println("RESIVED MSG TYPE 'waiting for transfer'")
			fmt.Println(msgInfoDownloadTask)

			//непосредственная передача файла
			go ReadSelectedFile(pfrdf, dfi, msgInfoDownloadTask.InfoFileDownloadTask.FileName)
		case "execute success":
			fmt.Println("RESIVED MSG TYPE 'execute success'")

			chanSendFile <- "success"

		case "execute failure":
			fmt.Println("RESIVED MSG TYPE 'execute failure'")

			chanSendFile <- "failure"
		}
	}
}
