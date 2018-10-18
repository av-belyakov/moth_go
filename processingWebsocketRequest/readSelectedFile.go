package processingWebsocketRequest

import (
	"fmt"
	"io"
	"math"
	"os"

	"moth_go/configure"
	"moth_go/errorMessage"
	"moth_go/helpers"
	"moth_go/saveMessageApp"
)

//ReadSelectedFile чтение выбранного файла
func ReadSelectedFile(pfrdf *configure.ParametrsFunctionRequestDownloadFiles, dfi *configure.DownloadFilesInformation, chanStopReadDownloadFiles <-chan configure.ChanStopReadDownloadFile) {
	fmt.Println("================= START function ReadSelectedFile... **********")

	const countByte = 1024
	fileName := dfi.RemoteIP[pfrdf.RemoteIP].FileInQueue.FileName

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

	//проверяем имя файла на соответствие регулярному выражению
	if err := helpers.CheckFileName(fileName, "fileName"); err != nil {

		fmt.Println("... ERROR function ReadSelectedFile - ", err)

		sendMessageError("unexpectedValue")

		return
	}

	filePath := dfi.RemoteIP[pfrdf.RemoteIP].DirectoryFiltering + "/" + fileName

	fmt.Println(filePath)

	fileStats, err := os.Stat(filePath)
	if err != nil {

		fmt.Println(err)

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

	countCycle := getCountCycle(fileStats.Size(), countByte)

	fmt.Println("COUNT CYCLE =", countCycle)

	var fileIsReaded error

	for i := 0; i <= countCycle; i++ {

		//fmt.Println("COUNT CYCLE =", countCycle, ", num:", i)

		select {
		case <-chanStopReadDownloadFiles:
			i = countCycle
			return

		case taskIndex := <-pfrdf.AccessClientsConfigure.ChanStopReadBinaryFile:
			//проверка наличия выполняющейся задачи с заданным ID и выход из функции
			if dfi.HasTaskDownloadFiles(pfrdf.RemoteIP, taskIndex) {
				dfi.DelTaskDownloadFiles(pfrdf.RemoteIP)

				pfrdf.AccessClientsConfigure.ChanInfoDownloadTaskSendMoth <- configure.ChanInfoDownloadTask{
					TaskIndex:      taskIndex,
					TypeProcessing: "cancel",
					RemoteIP:       pfrdf.RemoteIP,
				}

				fmt.Println("+++++++++++++++++++++++ resived msg STOP, send CANCEL for Flashlight")
			}

			return
		default:
			if fileIsReaded == io.EOF {
				return
			}

			data, err := readNextBytes(file, countByte, i)
			if err != nil {
				if err == io.EOF {
					pfrdf.AccessClientsConfigure.ChanWebsocketTranssmitionBinary <- data

					//последний набор байт информирующий Flashlight об окончании передачи файла
					pfrdf.AccessClientsConfigure.ChanWebsocketTranssmitionBinary <- []byte("file_EOF")

					fmt.Println("********* response MESSAGE TYPE 'execute completed' FOR FILE", fileName)

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

			pfrdf.AccessClientsConfigure.ChanWebsocketTranssmitionBinary <- data
		}
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
