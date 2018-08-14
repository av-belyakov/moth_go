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
func ProcessingUploadFiles(pfrdf *configure.ParametrsFunctionRequestDownloadFiles, dfi *configure.DownloadFilesInformation, chanSendFile <-chan configure.ChanSendFile) {
	fmt.Println("START function ProcessingUploadFiles...")

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

	//проверяем наличие файлов для передачи
	if len(dfi.RemoteIP[pfrdf.RemoteIP].ListDownloadFiles) == 0 {
		sendMessageError("filesNotFound")

		return
	}

	storageDirectory := dfi.RemoteIP[pfrdf.RemoteIP].DirectoryFiltering

	for fn := range dfi.RemoteIP[pfrdf.RemoteIP].ListDownloadFiles {
		fileStats, err := os.Stat(storageDirectory + "/" + fn)
		if err != nil {
			sendMessageError("filesNotFound")

			return
		}

		fileSize := fileStats.Size()

		if (fileSize != 24) && (!strings.Contains(fileStats.Name(), ".txt")) {
			fileHash, err := getHashSum(storageDirectory, fn)
			if err != nil {
				sendMessageError("filesNotFound")

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

			break
		}
	}

	for {
		fileTransmittion := <-chanSendFile

		switch fileTransmittion {
		case "success":
			//удачная передача файла, следующий файл

			fmt.Println("TRANSSMITION SUCCESS, NEXT FILE...")
		case "failure":
			//не удачная передача файла, повторная передача до обнуления счетчика

			fmt.Println("TRANSMITTION FAILURE, REPEATEDLY")
		}
	}
}
