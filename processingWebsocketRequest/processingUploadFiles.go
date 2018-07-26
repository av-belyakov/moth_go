package processingWebsocketRequest

import (
	"crypto/md5"
	"fmt"
	"io"
	"os"

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
		f, err := os.Open(pathFile + "/" + nameFile)
		if err != nil {
			return "", err
		}
		defer f.Close()

		h := md5.New()
		if _, err := io.Copy(h, f); err != nil {
			return "", err
		}

		return fmt.Sprintln(h.Sum(nil)), nil
	}

	//проверяем наличие файлов для передачи
	if len(dfi.RemoteIP[pfrdf.RemoteIP].ListDownloadFiles) == 0 {
		sendMessageError("filesNotFound")

		return
	}

	storageDirectory := dfi.RemoteIP[pfrdf.RemoteIP].DirectoryFiltering

	num := 0
	var firstFile string
	for fn := range dfi.RemoteIP[pfrdf.RemoteIP].ListDownloadFiles {
		if num >= 1 {
			return
		}

		fmt.Println("________________ ready send first file:", storageDirectory+"/"+fn)

		firstFile = fn
		num++
	}

	fileStats, err := os.Stat(storageDirectory + "/" + firstFile)
	if err != nil {
		sendMessageError("filesNotFound")

		return
	}

	fileHash, err := getHashSum(storageDirectory, firstFile)
	if err != nil {
		sendMessageError("filesNotFound")

		return
	}

	fmt.Println("------------************ FILE HASH", fileHash, " ----------------------")

	pfrdf.AccessClientsConfigure.ChanInfoDownloadTaskSendMoth <- configure.ChanInfoDownloadTask{
		TaskIndex:      dfi.RemoteIP[pfrdf.RemoteIP].TaskIndex,
		TypeProcessing: "execute",
		RemoteIP:       pfrdf.RemoteIP,
		InfoFileDownloadTask: configure.InfoFileDownloadTask{
			FileName: fileStats.Name(),
			FileHash: fileHash,
			FileSize: fileStats.Size(),
		},
	}

	for {
		fileTransmittion := <-chanSendFile

		switch fileTransmittion {
		case "success":
			//удачная передача файла, следующий файл

			fmt.Println("TRANSSMITION SUCCESS, NEXT FILE...")
		case "failure":
			//не удачная передача файла, повторная передача до бонуления счетчика

			fmt.Println("TRANSMITTION FAILURE, REPEATEDLY")
		}
	}
}
