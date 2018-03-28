package processingWebsocketRequest

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"moth_go/configure"
	"moth_go/errorMessage"
	"moth_go/helpers"
	"moth_go/saveMessageApp"
)

//CurrentListFilesFiltering содержит информацию о найденных файлах
type CurrentListFilesFiltering struct {
	Path       string
	Files      []string
	CountFiles int
	SizeFiles  int64
	ErrMsg     error
}

func searchFiles(result chan<- CurrentListFilesFiltering, disk string, currentTask *configure.InformationTaskFilter) {
	var currentListFilesFiltering CurrentListFilesFiltering
	currentListFilesFiltering.Path = disk
	currentListFilesFiltering.SizeFiles = 0

	fmt.Println("Search files for " + disk + " directory")

	if currentTask.UseIndexes {
		for _, file := range currentTask.ListFilesFilter[disk] {
			fileInfo, err := os.Stat(path.Join(disk, file))
			if err != nil {
				continue
			}

			currentListFilesFiltering.Files = append(currentListFilesFiltering.Files, file)
			currentListFilesFiltering.SizeFiles += fileInfo.Size()
			currentListFilesFiltering.CountFiles++
		}
	} else {
		files, err := ioutil.ReadDir(disk)
		if err != nil {
			currentListFilesFiltering.ErrMsg = err
			result <- currentListFilesFiltering
			return
		}

		for _, file := range files {
			fileIsUnixDate := file.ModTime().Unix()
			if currentTask.DateTimeStart < uint64(fileIsUnixDate) && uint64(fileIsUnixDate) < currentTask.DateTimeEnd {
				currentListFilesFiltering.Files = append(currentListFilesFiltering.Files, file.Name())
				currentListFilesFiltering.SizeFiles += file.Size()
				currentListFilesFiltering.CountFiles++
			}
		}
	}

	result <- currentListFilesFiltering
}

//подготовка списка файлов по которым будет выполнятся фильтрация
func getListFilesForFiltering(prf *configure.ParametrsFunctionRequestFilter, mft *configure.MessageTypeFilter) (int, int64) {

	fmt.Println("START function getListFilesForFiltering...")

	currentTask := prf.AccessClientsConfigure.Addresses[prf.RemoteIP].TaskFilter[mft.Info.TaskIndex]

	var fullCountFiles int
	var fullSizeFiles int64

	var result = make(chan CurrentListFilesFiltering, len(prf.CurrentDisks))
	defer close(result)

	var count int
	if currentTask.UseIndexes {
		count = len(currentTask.ListFilesFilter)
		for disk := range currentTask.ListFilesFilter {
			go searchFiles(result, disk, currentTask)
		}
	} else {
		count = len(prf.CurrentDisks)
		for _, disk := range prf.CurrentDisks {
			go searchFiles(result, disk, currentTask)
		}
	}

	for count > 0 {
		resultFoundFile := <-result

		if resultFoundFile.ErrMsg != nil {
			_ = saveMessageApp.LogMessage("error", fmt.Sprint(resultFoundFile.ErrMsg))
		}

		currentTask.ListFilesFilter[resultFoundFile.Path] = resultFoundFile.Files

		fullCountFiles += resultFoundFile.CountFiles
		fullSizeFiles += resultFoundFile.SizeFiles
		count--
	}

	fmt.Println("STOP function getListFilesForFiltering")

	return fullCountFiles, fullSizeFiles
}

//формируем шаблон для фильтрации\
func patternBashScript(mft *configure.MessageTypeFilter) string {
	fmt.Println(mft)
	return ""
}

//выполнение фильтрации
func filterProcessing(mft *configure.MessageTypeFilter) {
	fmt.Println(patternBashScript(mft))
}

func requestFilteringStart(prf *configure.ParametrsFunctionRequestFilter, mft *configure.MessageTypeFilter) {
	fmt.Println("function requestFilteringStart START...")
	fmt.Println("------------------- GET LIST FILES FOR FILTERING ---------------------")

	createDirectoryForFiltering := func() (err error) {
		dateTimeStart := time.Unix(1461929460, 0)
		dirName := strconv.Itoa(dateTimeStart.Year()) + "_" + dateTimeStart.Month().String() + "_" + strconv.Itoa(dateTimeStart.Day()) + "_" + strconv.Itoa(dateTimeStart.Hour()) + "_" + strconv.Itoa(dateTimeStart.Minute()) + "_" + mft.Info.TaskIndex

		filePath := path.Join(prf.PathStorageFilterFiles, "/", dirName)

		prf.AccessClientsConfigure.Addresses[prf.RemoteIP].TaskFilter[mft.Info.TaskIndex].DirectoryFiltering = filePath

		return os.MkdirAll(filePath, 0766)
	}

	createFileReadme := func() error {
		directoryName := prf.AccessClientsConfigure.Addresses[prf.RemoteIP].TaskFilter[mft.Info.TaskIndex].DirectoryFiltering
		fileOut, err := os.OpenFile(path.Join(directoryName, "/readme.txt"), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			return err
		}

		text := "The generated query is " + time.Now().String() + "\n\r"
		text += "\tclientIP: " + prf.RemoteIP + "\n"
		text += "\tserverIP: " + prf.ExternalIP + "\n"
		text += "\tsearch date start: " + time.Unix(int64(mft.Info.Settings.DateTimeStart), 0).String() + "\n"
		text += "\tsearch date end: " + time.Unix(int64(mft.Info.Settings.DateTimeEnd), 0).String() + "\n"
		text += "\tsearch ipaddress: " + strings.Join(mft.Info.Settings.IPAddress, "") + "\n"
		text += "\tsearch networks: " + strings.Join(mft.Info.Settings.Network, "") + "\n"

		writer := bufio.NewWriter(fileOut)
		defer func() {
			if err == nil {
				err = writer.Flush()
			}
		}()

		if _, err := writer.WriteString(text); err != nil {
			return err
		}

		return nil
	}

	//список файлов для фильтрации
	fullCountFiles, fullSizeFiles := getListFilesForFiltering(prf, mft)

	fmt.Println("full count files found: ", fullCountFiles)
	fmt.Println("full size files found: ", fullSizeFiles)

	if fullCountFiles == 0 {
		_ = saveMessageApp.LogMessage("info", "task ID: "+mft.Info.TaskIndex+", files needed to perform filtering not found")

		if err := errorMessage.SendErrorMessage(errorMessage.Options{
			RemoteIP:   prf.RemoteIP,
			ErrMsg:     "filesNotFound",
			TaskIndex:  mft.Info.TaskIndex,
			ExternalIP: prf.ExternalIP,
			Wsc:        prf.AccessClientsConfigure.Addresses[prf.RemoteIP].WsConnection,
		}); err != nil {
			_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
		}
		return
	}

	/* ТЕСТОВый БЛОК  */
	fmt.Println("TEST BLOCK...")
	for disk := range prf.AccessClientsConfigure.Addresses[prf.RemoteIP].TaskFilter[mft.Info.TaskIndex].ListFilesFilter {
		fmt.Println("DISK NAME: ", disk, " --- ", len(prf.AccessClientsConfigure.Addresses[prf.RemoteIP].TaskFilter[mft.Info.TaskIndex].ListFilesFilter[disk]), " files")
	}

	//создание директории для результатов фильтрации
	if err := createDirectoryForFiltering(); err != nil {
		_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))

		if err := errorMessage.SendErrorMessage(errorMessage.Options{
			RemoteIP:   prf.RemoteIP,
			ErrMsg:     "serverError",
			TaskIndex:  mft.Info.TaskIndex,
			ExternalIP: prf.ExternalIP,
			Wsc:        prf.AccessClientsConfigure.Addresses[prf.RemoteIP].WsConnection,
		}); err != nil {
			_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
		}
		return
	}

	//создание файла readme.txt
	if err := createFileReadme(); err != nil {
		_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
	}

	infoTaskFilter := *prf.AccessClientsConfigure.Addresses[prf.RemoteIP].TaskFilter[mft.Info.TaskIndex]
	//количество директорий для фильтрации
	infoTaskFilter.CountDirectoryFiltering = prf.AccessClientsConfigure.GetCountDirectoryFiltering(prf.RemoteIP, mft.Info.TaskIndex)
	//общее количество фильтруемых файлов
	infoTaskFilter.CountFilesFiltering = fullCountFiles
	//количество полных циклов
	infoTaskFilter.CountFullCycle = infoTaskFilter.CountFilesFiltering / infoTaskFilter.CountDirectoryFiltering
	//общий размер фильтруемых файлов
	infoTaskFilter.CountMaxFilesSize = fullSizeFiles

	var messageFilteringStart configure.MessageTypeFilteringStart
	messageFilteringStart.MessageType = "filtering"

	messageFilteringStart.Info.Processing = "start"
	messageFilteringStart.Info.TaskIndex = mft.Info.TaskIndex
	messageFilteringStart.Info.IPAddress = prf.ExternalIP
	messageFilteringStart.Info.DirectoryFiltering = infoTaskFilter.DirectoryFiltering
	messageFilteringStart.Info.CountDirectoryFiltering = infoTaskFilter.CountDirectoryFiltering
	messageFilteringStart.Info.CountFullCycle = infoTaskFilter.CountFullCycle
	messageFilteringStart.Info.CountFilesFiltering = infoTaskFilter.CountFilesFiltering
	messageFilteringStart.Info.CountMaxFilesSize = infoTaskFilter.CountMaxFilesSize
	messageFilteringStart.Info.CountCycleComplete = infoTaskFilter.CountCycleComplete
	messageFilteringStart.Info.CountFilesFound = infoTaskFilter.CountFilesFound
	messageFilteringStart.Info.CountFoundFilesSize = infoTaskFilter.CountFoundFilesSize
	messageFilteringStart.Info.CountFilesProcessed = infoTaskFilter.CountFilesProcessed
	messageFilteringStart.Info.CountFilesUnprocessed = infoTaskFilter.CountFilesUnprocessed
	messageFilteringStart.Info.ListFilesFilter = infoTaskFilter.ListFilesFilter

	formatJSON, err := json.Marshal(&messageFilteringStart)
	if err != nil {
		_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
	}

	fmt.Println("SENDING TO CHAN JSON OBJECT")

	if err := prf.AccessClientsConfigure.Addresses[prf.ExternalIP].WsConnection.WriteMessage(1, formatJSON); err != nil {
		_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
	}

	fmt.Println("WEBSOCKET ERROR: ", err)

	fmt.Println("***************** STOP requestFilteringStart ************")

	filterProcessing(mft)
}

/*
FilterinInfoPattern
	FilterCountPattern
	DirectoryFiltering      string              `json:"directoryFiltering"`
	CountDirectoryFiltering int                 `json:"CountDirectoryFiltering"`
	CountFullCycle          int                 `json:"countFullCycle"`
	CountFilesFiltering     int                 `json:"countFilesFiltering"`
	CountMaxFilesSize       int64                 `json:"countMaxFilesSize"`
	UseIndexes              bool                `json:"useIndexes"`
	ListFilesFilter         map[string][]string `json:"listFilesFilter"`

type FilterCountPattern struct {
	CountCycleComplete    int   `json:"countCycleComplete"`
	CountFilesFound       int   `json:"countFilesFound"`
	CountFoundFilesSize   int64 `json:"countFoundFilesSize"`
	CountFilesProcessed   int   `json:"countFilesProcessed"`
	CountFilesUnprocessed int   `json:"countFilesUnprocessed"`
}

*/

/*
type InformationTaskFilter struct {
	DateTimeStart, DateTimeEnd uint64
	IPAddress, Network         []string
	UseIndexes                 bool
	  DirectoryFiltering         string
	  CountDirectoryFiltering    int
	  CountFullCycle             int
	CountCycleComplete         int 0
      CountFilesFiltering        int
	CountFilesFound            int 0
	CountFilesProcessed        int 0
      CountMaxFilesSize          uint64
	CountFoundFilesSize        uint64 0
	ListFilesFilter            map[string][]string
}
*/

func requestFilteringStop(prf *configure.ParametrsFunctionRequestFilter, mft *configure.MessageTypeFilter) {

	fmt.Println("function requestFilteringStop START...")

	var messageFilteringStop configure.MessageTypeFilteringStop
	messageFilteringStop.MessageType = "filtering"
	messageFilteringStop.Info.Processing = "stop"
	messageFilteringStop.Info.TaskIndex = mft.Info.TaskIndex
	messageFilteringStop.Info.IPAddress = prf.ExternalIP

	formatJSON, err := json.Marshal(&messageFilteringStop)
	if err != nil {
		_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
	}

	if err := prf.AccessClientsConfigure.Addresses[prf.ExternalIP].WsConnection.WriteMessage(1, formatJSON); err != nil {
		_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
	}

	delete(prf.AccessClientsConfigure.Addresses[prf.RemoteIP].TaskFilter, mft.Info.TaskIndex)
}

func requestFilteringResume(prf *configure.ParametrsFunctionRequestFilter, mft *configure.MessageTypeFilter) {
	fmt.Println("function requestFilteringResume START...")

}

//RequestTypeFilter обрабатывает запросы связанные с фильтрацией
func RequestTypeFilter(prf *configure.ParametrsFunctionRequestFilter, mtf *configure.MessageTypeFilter) {

	fmt.Println("\nFILTERING: function RequestTypeFilter STARTING...")

	//проверяем количество одновременно выполняемых задач
	if prf.AccessClientsConfigure.IsMaxCountProcessFiltering(prf.RemoteIP) {

		fmt.Println("TEST count tasks")

		if err := errorMessage.SendErrorMessage(errorMessage.Options{
			RemoteIP:   prf.RemoteIP,
			ErrMsg:     "limitTasks",
			TaskIndex:  mtf.Info.TaskIndex,
			ExternalIP: prf.ExternalIP,
			Wsc:        prf.AccessClientsConfigure.Addresses[prf.RemoteIP].WsConnection,
		}); err != nil {
			_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
		}
		return
	}

	//проверка полученных от пользователя данных (список адресов, сетей и файлов по которым должна быть выполненна фильтрация)
	if errMsg, ok := helpers.InputParametrsForFiltering(prf, mtf); !ok {

		fmt.Println("TEST check user data")

		if err := errorMessage.SendErrorMessage(errorMessage.Options{
			RemoteIP:   prf.RemoteIP,
			ErrMsg:     errMsg,
			TaskIndex:  mtf.Info.TaskIndex,
			ExternalIP: prf.ExternalIP,
			Wsc:        prf.AccessClientsConfigure.Addresses[prf.RemoteIP].WsConnection,
		}); err != nil {
			_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
		}
		return
	}

	fmt.Println("------------------- VERIFY DATA FOR FILTERING ---------------------")
	//fmt.Printf("%#v\n", prf.AccessClientsConfigure.Addresses[prf.RemoteIP].TaskFilter[prf.MessageTypeFilter.Info.TaskIndex].ListFilesFilter)

	typeRequest := mtf.Info.Processing
	switch typeRequest {
	case "on":
		requestFilteringStart(prf, mtf)
	case "off":
		requestFilteringStop(prf, mtf)
	case "resume":
		requestFilteringResume(prf, mtf)
	}
}
