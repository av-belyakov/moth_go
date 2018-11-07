package processingWebsocketRequest

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"

	"moth_go/configure"
	"moth_go/errorMessage"
	"moth_go/saveMessageApp"
)

//ProcessingUploadFiles выполняет передачу информации о файлах
func ProcessingUploadFiles(chanFinishDownloadFiles chan<- struct{}, pfrdf *configure.ParametrsFunctionRequestDownloadFiles, dfi *configure.DownloadFilesInformation, chanSendFile <-chan configure.ChanSendFile) {
	fmt.Println("START function ProcessingUploadFiles...")

	storageDirectory := dfi.RemoteIP[pfrdf.RemoteIP].DirectoryFiltering

	sendMessageError := func(errType string) {
		if err := errorMessage.SendErrorMessage(errorMessage.Options{
			RemoteIP:   pfrdf.RemoteIP,
			ErrMsg:     errType,
			TaskIndex:  dfi.RemoteIP[pfrdf.RemoteIP].TaskIndex,
			ExternalIP: pfrdf.ExternalIP,
			Wsc:        pfrdf.AccessClientsConfigure.Addresses[pfrdf.RemoteIP].WsConnection,
		}); err != nil {
			_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
		}
		return
	}

	getHashSum := func(pathFile, nameFile string) (string, error) {

		fmt.Println("READ FILE", pathFile+"/"+nameFile)

		f, err := os.Open(pathFile + "/" + nameFile)
		if err != nil {
			return "", err
		}
		defer f.Close()

		h := md5.New()
		if _, err := io.Copy(h, f); err != nil {
			return "", err
		}

		return hex.EncodeToString(h.Sum(nil)), nil
	}

	sendMessageExecuteFile := func() {

		fmt.Println("......... start function sendMessageExecuteFile ..........")

		//проверяем наличие файлов для передачи
		if len(dfi.RemoteIP[pfrdf.RemoteIP].ListDownloadFiles) == 0 {

			chanFinishDownloadFiles <- struct{}{}

			return
		}

		for fn := range dfi.RemoteIP[pfrdf.RemoteIP].ListDownloadFiles {
			fileStats, err := os.Stat(storageDirectory + "/" + fn)
			if err != nil {

				fmt.Println("*********** file", fn, "is not file 111")

				sendMessageError("filesNotFound")
				return
			}

			fileSize := fileStats.Size()

			if (fileSize > 24) && (!strings.Contains(fileStats.Name(), ".txt")) {
				fileHash, err := getHashSum(storageDirectory, fn)
				if err != nil {

					fmt.Println("*********** file", fn, "is not file 222")

					sendMessageError("filesNotFound")
					return
				}

				//для того что бы остановить передачу когда соединение было разорванно
				if found := dfi.HasRemoteIPDownloadFiles(pfrdf.RemoteIP); !found {
					return
				}

				dfi.RemoteIP[pfrdf.RemoteIP].FileInQueue = configure.FileInfoinQueue{
					FileName: fn,
					FileHash: fileHash,
					FileSize: fileSize,
				}

				fmt.Println("------------************ FILE HASH", fileHash, " ----------------------")

				pfrdf.AccessClientsConfigure.ChanInfoDownloadTaskSendMoth <- configure.ChanInfoDownloadTask{
					TaskIndex:      dfi.RemoteIP[pfrdf.RemoteIP].TaskIndex,
					TypeProcessing: "execute",
					RemoteIP:       pfrdf.RemoteIP,
					InfoFileDownloadTask: configure.InfoFileDownloadTask{
						FileName: fileStats.Name(),
						FileHash: fileHash,
						FileSize: fileSize,
					},
				}

				return
			}
		}
	}

	//отправляем сообщение о готовности к передаче файла или сообщение о завершении передачи
	sendMessageExecuteFile()

	fmt.Println("____ function processingUploadFiles ____, before processing chanel 'chanSendFile'")

	for resultTransmittion := range chanSendFile {

		fmt.Println("========================= recived message is chan chanSendFile ", resultTransmittion, "==========================")

		filePath := storageDirectory + "/" + dfi.RemoteIP[pfrdf.RemoteIP].FileInQueue.FileName

		switch resultTransmittion {
		case "success":
			/* удачная передача файла, следующий файл */

			fmt.Println("TRANSSMITION SUCCESS, NEXT FILE...")

			//удаляем уже переданный файл из списка dfi.RemoteIP[pfrdf.RemoteIP].ListDownloadFiles
			dfi.RemoveFileFromListFiles(pfrdf.RemoteIP, dfi.RemoteIP[pfrdf.RemoteIP].FileInQueue.FileName)

			//удаляем непосредственно сам файл
			if err := os.Remove(filePath); err != nil {
				_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
			}

			//отправляем сообщение о готовности к передаче следующего файла или сообщение о завершении передачи
			sendMessageExecuteFile()

		case "failure":
			/* повторная передача до обнуления счетчика */

			fmt.Println("TRANSMITTION FAILURE, REPEATEDLY")

			fileName := dfi.RemoteIP[pfrdf.RemoteIP].FileInQueue.FileName
			if _, ok := dfi.RemoteIP[pfrdf.RemoteIP].ListDownloadFiles[fileName]; !ok {
				sendMessageError("filesNotFound")
			}

			/*if dfi.RemoteIP[pfrdf.RemoteIP].ListDownloadFiles[fileName].NumberTransferAttempts > 0 {
				dfi.RemoteIP[pfrdf.RemoteIP].ListDownloadFiles[fileName].NumberTransferAttempts -= dfi.RemoteIP[pfrdf.RemoteIP].ListDownloadFiles[fileName].NumberTransferAttempts

				FileInQueue := dfi.RemoteIP[pfrdf.RemoteIP].FileInQueue

				pfrdf.AccessClientsConfigure.ChanInfoDownloadTaskSendMoth <- configure.ChanInfoDownloadTask{
					TaskIndex:      dfi.RemoteIP[pfrdf.RemoteIP].TaskIndex,
					TypeProcessing: "execute",
					RemoteIP:       pfrdf.RemoteIP,
					InfoFileDownloadTask: configure.InfoFileDownloadTask{
						FileName: FileInQueue.FileName,
						FileHash: FileInQueue.FileHash,
						FileSize: FileInQueue.FileSize,
					},
				}

				return
			}*/

			if dfi.RemoteIP[pfrdf.RemoteIP].ListDownloadFiles[fileName].NumberTransferAttempts > 0 {
				//удаляем уже переданный файл из списка dfi.RemoteIP[pfrdf.RemoteIP].ListDownloadFiles
				dfi.RemoveFileFromListFiles(pfrdf.RemoteIP, dfi.RemoteIP[pfrdf.RemoteIP].FileInQueue.FileName)

				//удаляем непосредственно сам файл
				if err := os.Remove(filePath); err != nil {
					_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
				}
			}

			//отправляем сообщение о готовности к передаче следующего файла или сообщение о завершении передачи
			sendMessageExecuteFile()
		}
	}

	fmt.Println("Останов процесса выгрузки файлов, функция ProcessingUploadFiles ++++")

}
