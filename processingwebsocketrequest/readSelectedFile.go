package processingwebsocketrequest

import (
	"fmt"
	"io"
	"math"
	"os"

	"moth_go/configure"
	"moth_go/errormessage"
	"moth_go/helpers"
	"moth_go/savemessageapp"
)

//ReadSelectedFile чтение выбранного файла
func ReadSelectedFile(pfrdf *configure.ParametrsFunctionRequestDownloadFiles, dfi *configure.DownloadFilesInformation) {
	//инициализируем функцию конструктор для записи лог-файлов
	saveMessageApp := savemessageapp.New()

	const countByte = 1024
	fileName := dfi.RemoteIP[pfrdf.RemoteIP].FileInQueue.FileName

	sendMessageError := func(errType string) {
		if err := errormessage.SendErrorMessage(errormessage.Options{
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

	//проверяем имя файла на соответствие регулярному выражению
	if err := helpers.CheckFileName(fileName, "fileName"); err != nil {
		sendMessageError("unexpectedValue")

		return
	}

	filePath := dfi.RemoteIP[pfrdf.RemoteIP].DirectoryFiltering + "/" + fileName

	fileStats, err := os.Stat(filePath)
	if err != nil {
		sendMessageError("filesNotFound")
		_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))

		return
	}

	if fileStats.Size() <= 24 {
		sendMessageError("filesNotFound")
		_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))

		return
	}

	file, err := os.Open(filePath)
	if err != nil {
		sendMessageError("filesNotFound")
		_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))

		return
	}
	defer file.Close()

	var fileIsReaded error

	strHash := []byte(dfi.RemoteIP[pfrdf.RemoteIP].FileInQueue.FileHash)
	countCycle := getCountCycle(fileStats.Size(), countByte)

	for i := 0; i <= countCycle; i++ {
		if fileIsReaded == io.EOF {
			return
		}

		data, err := readNextBytes(file, countByte, i)
		if err != nil {
			if err == io.EOF {
				data = append(strHash, data...)
				pfrdf.AccessClientsConfigure.ChanWebsocketTranssmitionBinary <- data

				newStrHash := append(strHash, []byte(" moth say: file_EOF")...)
				//последний набор байт информирующий Flashlight об окончании передачи файла
				pfrdf.AccessClientsConfigure.ChanWebsocketTranssmitionBinary <- newStrHash

				if found := dfi.HasRemoteIPDownloadFiles(pfrdf.RemoteIP); !found {
					return
				}

				pfrdf.AccessClientsConfigure.ChanInfoDownloadTaskSendMoth <- configure.ChanInfoDownloadTask{
					TaskIndex:      dfi.RemoteIP[pfrdf.RemoteIP].TaskIndex,
					TypeProcessing: "execute completed",
					RemoteIP:       pfrdf.RemoteIP,
					InfoFileDownloadTask: configure.InfoFileDownloadTask{
						FileName: fileName,
						FileSize: fileStats.Size(),
					},
				}

				fileIsReaded = io.EOF
			} else {
				sendMessageError("filesNotFound")
				_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
			}

			return
		}

		data = append(strHash, data...)
		pfrdf.AccessClientsConfigure.ChanWebsocketTranssmitionBinary <- data
	}
}

func readNextBytes(file *os.File, number, nextNum int) ([]byte, error) {
	bytes := make([]byte, number)
	var off int64

	if nextNum != 0 {
		off = int64(number * nextNum)
	}

	rb, err := file.ReadAt(bytes, off)
	if err != nil {
		if err == io.EOF {
			return bytes[:rb], err
		}

		return nil, err
	}

	return bytes, nil
}

func getCountCycle(fileSize int64, countByte int) int {
	newFileSize := float64(fileSize)
	newCountByte := float64(countByte)
	x := math.Floor(newFileSize / newCountByte)
	y := newFileSize / newCountByte

	if (y - x) != 0 {
		x++
	}

	return int(x)
}
