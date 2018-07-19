package processingWebsocketRequest

import (
	"fmt"
	"moth_go/configure"
)

//ProcessingDownloadFiles осуществляет обработку запросов на скачивание файлов
func ProcessingDownloadFiles(pfrdf *configure.ParametrsFunctionRequestDownloadFiles, dfi *configure.DownloadFilesInformation) {
	fmt.Println("DOWNLOADING, function ProcessingDownloadFiles START...")

	for {
		msgInfoDownloadTask := <-pfrdf.AccessClientsConfigure.ChanInfoDownloadTaskGetMoth

		fmt.Println("func ProcessingDownloadFiles package routeWebSocketRequest")
		fmt.Println(msgInfoDownloadTask)

	}
	/*switch mtdf.Info.Processing {
	case "start":

	case "stop":

	case "ready":

	case "waiting for transfer":

	case "execute success":

	case "execute failure":

	}*/
}
