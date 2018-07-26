package helpers

import (
	"errors"
	"fmt"
	"os"

	"moth_go/configure"
	"moth_go/errorMessage"
	"moth_go/saveMessageApp"
)

//MergingFileListForTaskDownloadFiles выполняет объединение списков файлов переданных клиентом и предназначенны для выгрузки файлов
func MergingFileListForTaskDownloadFiles(pfrdf *configure.ParametrsFunctionRequestDownloadFiles, mtdf configure.MessageTypeDownloadFiles, dfi *configure.DownloadFilesInformation) (bool, error) {
	errorMsg := errorMessage.Options{
		RemoteIP:   pfrdf.RemoteIP,
		ErrMsg:     "filesNotFound",
		TaskIndex:  mtdf.Info.TaskIndex,
		ExternalIP: pfrdf.ExternalIP,
		Wsc:        pfrdf.AccessClientsConfigure.Addresses[pfrdf.RemoteIP].WsConnection,
	}

	if !mtdf.Info.DownloadSelectedFiles {
		return true, errors.New("no files were selected")
	}

	if mtdf.Info.NumberMessageParts[0] == 0 {
		dfi.RemoteIP[pfrdf.RemoteIP].TotalCountDownloadFiles = mtdf.Info.CountDownloadSelectedFiles
		dfi.RemoteIP[pfrdf.RemoteIP].SelectedFiles = true

		dfi.RemoteIP[pfrdf.RemoteIP].NumberPleasantMessages = 0

		return false, nil
	}

	for _, fileName := range mtdf.Info.ListDownloadSelectedFiles {

		fmt.Println("PATH TO FILE =", dfi.RemoteIP[pfrdf.RemoteIP].DirectoryFiltering+"/"+fileName)

		f, err := os.OpenFile(dfi.RemoteIP[pfrdf.RemoteIP].DirectoryFiltering+"/"+fileName, os.O_RDONLY, 0666)
		if err != nil {
			if err := errorMessage.SendErrorMessage(errorMsg); err != nil {
				_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
			}
			continue
		}

		if _, err := f.Stat(); err != nil {
			if err := errorMessage.SendErrorMessage(errorMsg); err != nil {
				_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
			}
			continue
		}

		dfi.RemoteIP[pfrdf.RemoteIP].ListDownloadFiles[fileName] = &configure.FileInformationDownloadFiles{
			NumberTransferAttempts: 3,
		}

		f.Close()
	}

	dfi.RemoteIP[pfrdf.RemoteIP].NumberPleasantMessages++

	if dfi.RemoteIP[pfrdf.RemoteIP].NumberPleasantMessages == mtdf.Info.NumberMessageParts[1] {

		fmt.Println("!!!!!! LAST ELEMENT DOWNLOAD FILES")
		fmt.Println(dfi.RemoteIP[pfrdf.RemoteIP].NumberPleasantMessages, " = ", mtdf.Info.NumberMessageParts[1])

		//проверяем количество полученных имен файлов с общим количеством в TotalCountDownloadFiles
		if dfi.RemoteIP[pfrdf.RemoteIP].TotalCountDownloadFiles != len(dfi.RemoteIP[pfrdf.RemoteIP].ListDownloadFiles) {
			_ = saveMessageApp.LogMessage("error", "the number of files transferred does not match the number specified in the TotalCountDownloadFiles")

			return true, nil
			//return true, errors.New("the number of files transferred does not match the number specified in the TotalCountDownloadFiles")
		}

		return true, nil
	}

	return false, nil
}
