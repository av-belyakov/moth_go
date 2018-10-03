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

	//канал информирующий об остановке передачи файлов
	chanSendStopDownloadFiles := make(chan configure.ChanSendStopDownloadFiles)

	for msgInfoDownloadTask := range pfrdf.AccessClientsConfigure.ChanInfoDownloadTaskGetMoth {
		fmt.Println("****ROUTING**** func ProcessingDownloadFiles package routeWebSocketRequest")

		switch msgInfoDownloadTask.TypeProcessing {
		case "stop":
			fmt.Println("RESIVED MSG TYPE 'stop'", "drop user type 'DownloadFilesInformation'")

			pfrdf.AccessClientsConfigure.ChanInfoDownloadTaskSendMoth <- configure.ChanInfoDownloadTask{
				TaskIndex:      dfi.RemoteIP[pfrdf.RemoteIP].TaskIndex,
				TypeProcessing: "stop",
				RemoteIP:       pfrdf.RemoteIP,
			}

			//останавливаем выгрузку файлов
			chanSendStopDownloadFiles <- struct{}{}

			//удаляем запись о выполнявшейся задаче
			dfi.DelTaskDownloadFiles(pfrdf.RemoteIP)

			//выход из цикла, завершение go-подпрограммы
			return

		/*case "cancel":
		fmt.Println("RESIVED MSG TYPE 'cancel'")

		//выход из цикла, завершение go-подпрограммы
		return*/

		case "ready":
			fmt.Println("RESIVED MSG TYPE 'ready'")

			//инициализация начала передачи файлов
			go ProcessingUploadFiles(pfrdf, dfi, chanSendFile, chanSendStopDownloadFiles)

		case "waiting for transfer":
			fmt.Println("RESIVED MSG TYPE 'waiting for transfer'")
			fmt.Println(msgInfoDownloadTask)

			//непосредственная передача файла
			go ReadSelectedFile(pfrdf, dfi)

		case "execute success":
			fmt.Println("***** RESIVED MSG TYPE 'execute success' =====")

			chanSendFile <- "success"

		case "execute failure":
			fmt.Println("***** RESIVED MSG TYPE 'execute failure' =====")

			chanSendFile <- "failure"

		}
	}

	/*for {
	msgInfoDownloadTask := <-pfrdf.AccessClientsConfigure.ChanInfoDownloadTaskGetMoth

	fmt.Println("****ROUTING**** func ProcessingDownloadFiles package routeWebSocketRequest")

	switch msgInfoDownloadTask.TypeProcessing {

	/*
		!!! ПОКА НЕ РЕШИЛ КАКОЙ КОМАНДОЙ ОСТАНОВИТЬ ЗАДАЧУ ПО ВЫГРУЗКЕ ФАЙЛОВ ('stop' или 'cancel') !!!
	*/

	/*case "stop":
			fmt.Println("RESIVED MSG TYPE 'stop'", "drop user type 'DownloadFilesInformation'")

			//необходимо остановить выгрузку файлов и удалить запись о задаче
			dfi.DelTaskDownloadFiles(pfrdf.RemoteIP)

			chanSendStopDownloadFiles <- struct{}{}

			//выход из цикла, завершение go-подпрограммы
			return

		case "cancel":
			fmt.Println("RESIVED MSG TYPE 'cancel'")
			//необходимо остановить выгрузку файлов и удалить запись о задаче
			dfi.DelTaskDownloadFiles(pfrdf.RemoteIP)

			chanSendStopDownloadFiles <- struct{}{}

			fmt.Println(dfi)

			//выход из цикла, завершение go-подпрограммы
			return

		case "ready":
			fmt.Println("RESIVED MSG TYPE 'ready'")

			//инициализация начала передачи файлов
			go ProcessingUploadFiles(pfrdf, dfi, chanSendFile, chanSendStopDownloadFiles)

		case "waiting for transfer":
			fmt.Println("RESIVED MSG TYPE 'waiting for transfer'")
			fmt.Println(msgInfoDownloadTask)

			//непосредственная передача файла
			go ReadSelectedFile(pfrdf, dfi)

		case "execute success":
			fmt.Println("***** RESIVED MSG TYPE 'execute success' =====")

			chanSendFile <- "success"

		case "execute failure":
			fmt.Println("***** RESIVED MSG TYPE 'execute failure' =====")

			chanSendFile <- "failure"

		}
	}*/
}
