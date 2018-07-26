package processingWebsocketRequest

import (
	"fmt"
	"moth_go/configure"
)

//RouteProcessingDownloadFiles осуществляет обработку запросов на скачивание файлов
func RouteProcessingDownloadFiles(pfrdf *configure.ParametrsFunctionRequestDownloadFiles, dfi *configure.DownloadFilesInformation) {
	fmt.Println("*************** DOWNLOADING, function ProcessingDownloadFiles START...")
	fmt.Println("--- 2 /////////////////////// dfi listFiles ", dfi.RemoteIP[pfrdf.RemoteIP].ListDownloadFiles)

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

		case "waiting for transfer":
			fmt.Println("RESIVED MSG TYPE 'waiting for transfer'")

		case "execute success":
			fmt.Println("RESIVED MSG TYPE 'execute success'")

		case "execute failure":
			fmt.Println("RESIVED MSG TYPE 'execute failure'")

		}
	}
}
