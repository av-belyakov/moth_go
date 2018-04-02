package processingWebsocketRequest

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
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

//PatternParametersFiltering содержит данные необходимые для подготовки шаблона
type PatternParametersFiltering struct {
	ParameterFilter        *configure.MessageTypeFilter
	DirectoryName          string
	TypeAreaNetwork        int
	PathStorageFilterFiles string
	ListFiles              []string
}

//FormingMessageFilterComplete содержит детали для отправки сообщения о завершении фильтрации
type FormingMessageFilterComplete struct {
	TaskIndex      string
	RemoteIP       string
	CountDirectory int
	Done           chan string
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

//формируем шаблон для фильтрации
func patternBashScript(patternParametersFiltering *PatternParametersFiltering) string {
	getIPAddressString := func(ipaddreses []string) (searchHosts string) {
		num := 0
		if len(ipaddreses) != 0 {
			if len(ipaddreses) == 1 {
				searchHosts += " host " + ipaddreses[0]
			} else {
				for _, ip := range ipaddreses {
					searchHosts += " host " + ip
					if num < (len(ipaddreses) - 1) {
						searchHosts += " or"
					}
					num++
				}
			}
		}
		return searchHosts
	}

	getNetworksString := func(networks []string) (searchNetworks string) {
		num := 0
		if len(networks) != 0 {
			if len(networks) == 1 {
				searchNetworks += " net " + networks[0]
			} else {
				for _, net := range networks {
					searchNetworks += " host " + net
					if num < (len(networks) - 1) {
						searchNetworks += " or"
					}
					num++
				}
			}
		}
		return searchNetworks
	}

	fmt.Println("function patternBashScript START...")

	bind := " "
	//формируем строку для поиска хостов
	searchHosts := getIPAddressString(patternParametersFiltering.ParameterFilter.Info.Settings.IPAddress)

	//формируем строку для поиска сетей
	searchNetwork := getNetworksString(patternParametersFiltering.ParameterFilter.Info.Settings.Network)

	if len(patternParametersFiltering.ParameterFilter.Info.Settings.IPAddress) > 0 && len(patternParametersFiltering.ParameterFilter.Info.Settings.Network) > 0 {
		bind = " or"
	}

	listTypeArea := map[int]string{
		1: "",
		2: " '(vlan or ip)' and ",
		3: " '(pppoes && ip)' and ",
	}

	pattern := " tcpdump -r " + patternParametersFiltering.DirectoryName + "/$files"
	pattern += listTypeArea[patternParametersFiltering.TypeAreaNetwork] + searchHosts + bind + searchNetwork
	pattern += " -w " + patternParametersFiltering.PathStorageFilterFiles + "/$files;"
	pattern += " echo 'completed:" + patternParametersFiltering.DirectoryName + "';"

	return pattern
}

//выполнение фильтрации
func filterProcessing(done chan<- string, patternParametersFiltering *PatternParametersFiltering, patternBashScript string, prf *configure.ParametrsFunctionRequestFilter) {
	listFilesFilter := patternParametersFiltering.ListFiles
	task := prf.AccessClientsConfigure.Addresses[prf.RemoteIP].TaskFilter[patternParametersFiltering.ParameterFilter.Info.TaskIndex]

	var mtfeou configure.MessageTypeFilteringExecutedOrUnexecuted
	mtfeou.MessageType = "filtering"
	mtfeou.Info.ProcessingFile.DirectoryLocation = patternParametersFiltering.DirectoryName
	mtfeou.Info.IPAddress = prf.RemoteIP
	mtfeou.Info.TaskIndex = patternParametersFiltering.ParameterFilter.Info.TaskIndex

	for _, file := range listFilesFilter {
		mtfeou.Info.ProcessingFile.FileName = file

		newPatternBashScript := strings.Replace(patternBashScript, "$files", file, -1)

		//		_, err := exec.Command("sh", "-c", newPatternBashScript).Output()
		err := exec.Command("sh", "-c", newPatternBashScript).Run()

		if err != nil {
			task.CountFilesUnprocessed += task.CountFilesUnprocessed
			mtfeou.Info.CountFilesUnprocessed = task.CountFilesUnprocessed
			mtfeou.Info.Processing = "unexecuted"
			mtfeou.Info.ProcessingFile.StatusProcessed = false
		} else {
			task.CountFilesProcessed += task.CountFilesProcessed
			mtfeou.Info.CountFilesProcessed = task.CountFilesProcessed
			mtfeou.Info.Processing = "executed"
			mtfeou.Info.ProcessingFile.StatusProcessed = true
		}

		//получаем количество найденных файлов и их размер
		countFiles, fullSizeFiles, err := countNumberFilesFound(patternParametersFiltering.PathStorageFilterFiles)
		if err != nil {
			_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
		}

		task.CountCycleComplete += task.CountCycleComplete

		mtfeou.Info.CountCycleComplete = task.CountCycleComplete
		mtfeou.Info.CountFilesFound = countFiles
		mtfeou.Info.CountFoundFilesSize = fullSizeFiles

		fmt.Println(newPatternBashScript)
		//		fmt.Printf("%#v", mtfeou)

		formatJSON, err := json.Marshal(&mtfeou)
		if err != nil {
			_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
		}

		if err := prf.AccessClientsConfigure.Addresses[prf.RemoteIP].SendWsMessage(1, formatJSON); err != nil {
			_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
		}

		/*		if err := prf.AccessClientsConfigure.Addresses[prf.RemoteIP].WsConnection.WriteMessage(1, formatJSON); err != nil {
				_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
			}*/
	}
	done <- patternParametersFiltering.DirectoryName

	/*
			type MessageTypeFilteringExecutedOrUnexecuted struct {
			 MessageType string                                     `json:"messageType"`
			Info        MessageTypeFilteringExecuteOrUnexecuteInfo `json:"info"`
		}

		type MessageTypeFilteringExecuteOrUnexecuteInfo struct {
			FilterinInfoPattern
			FilterCountPattern
			ProcessingFile InfoProcessingFile `json:"infoProcessingFile"`
			//	UnprocessingFile string `json:"unprocessingFile"`
		}

		type InfoProcessingFile struct {
			 FileName          string `json:"fileName"`
			 DirectoryLocation string `json:"directoryLocation"`
			 StatusProcessed   bool `json:"statusProcessed"`
		}

		type FilterinInfoPattern struct {
			Processing string `json:"processing"`
			 TaskIndex  string `json:"taskIndex"`
			 IPAddress  string `json:"ipAddress"`
		}

		type FilterCountPattern struct {
			CountCycleComplete    int   `json:"countCycleComplete"`
			CountFilesFound       int   `json:"countFilesFound"`
			CountFoundFilesSize   int64 `json:"countFoundFilesSize"`
			CountFilesProcessed   int   `json:"countFilesProcessed"`
			CountFilesUnprocessed int   `json:"countFilesUnprocessed"`
		}

	*/
}

//подстчет количества найденных файлов
func countNumberFilesFound(directoryResultFilter string) (count int, size int64, err error) {
	files, err := ioutil.ReadDir(directoryResultFilter)
	if err != nil {
		return count, size, err
	}

	for _, file := range files {
		size += file.Size()
	}

	return (count - 1), size, nil
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

	filteringComplete := func(fmfc *FormingMessageFilterComplete, prf *configure.ParametrsFunctionRequestFilter) {
		var dirComplete, countCyclesComplete int
		var dirNameComplete string

		for dirComplete < fmfc.CountDirectory {
			dirNameComplete = <-fmfc.Done
			if len(dirNameComplete) > 0 {
				dirComplete++
			}
		}

		fmt.Println("--- FILTERING COMPLITE --- directory filtering is ", fmfc.CountDirectory)

		close(fmfc.Done)

		messageTypeFilteringComplete := configure.MessageTypeFilteringComplete{
			"filtering",
			configure.MessageTypeFilteringCompleteInfo{
				FilterinInfoPattern: configure.FilterinInfoPattern{
					Processing: "complete",
					TaskIndex:  fmfc.TaskIndex,
					IPAddress:  fmfc.RemoteIP,
				},
				CountCycleComplete: countCyclesComplete,
			},
		}

		formatJSON, err := json.Marshal(&messageTypeFilteringComplete)
		if err != nil {
			_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
		}

		fmt.Println("SENDING TO CHAN JSON OBJECT COMPLETE")

		if err := prf.AccessClientsConfigure.Addresses[prf.RemoteIP].SendWsMessage(1, formatJSON); err != nil {
			_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
		}
		/*if err := prf.AccessClientsConfigure.Addresses[prf.RemoteIP].WsConnection.WriteMessage(1, formatJSON); err != nil {
			_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
		}*/

		//удаляем задачу
		delete(prf.AccessClientsConfigure.Addresses[fmfc.RemoteIP].TaskFilter, fmfc.TaskIndex)
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
	infoTaskFilter.CountFullCycle = infoTaskFilter.CountFilesFiltering
	//общий размер фильтруемых файлов
	infoTaskFilter.CountMaxFilesSize = fullSizeFiles

	messageFilteringStart := configure.MessageTypeFilteringStart{
		"filtering",
		configure.MessageTypeFilteringStartInfo{
			configure.FilterinInfoPattern{
				Processing: "start",
				TaskIndex:  mft.Info.TaskIndex,
				IPAddress:  prf.ExternalIP,
			},
			configure.FilterCountPattern{
				CountCycleComplete:    infoTaskFilter.CountCycleComplete,
				CountFilesFound:       infoTaskFilter.CountFilesFound,
				CountFoundFilesSize:   infoTaskFilter.CountFoundFilesSize,
				CountFilesProcessed:   infoTaskFilter.CountFilesProcessed,
				CountFilesUnprocessed: infoTaskFilter.CountFilesUnprocessed,
			},
			infoTaskFilter.DirectoryFiltering,
			infoTaskFilter.CountDirectoryFiltering,
			infoTaskFilter.CountFullCycle,
			infoTaskFilter.CountFilesFiltering,
			infoTaskFilter.CountMaxFilesSize,
			false,
			infoTaskFilter.ListFilesFilter,
		},
	}

	formatJSON, err := json.Marshal(&messageFilteringStart)
	if err != nil {
		_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
	}

	fmt.Println("SENDING TO CHAN JSON OBJECT")

	if err := prf.AccessClientsConfigure.Addresses[prf.RemoteIP].SendWsMessage(1, formatJSON); err != nil {
		_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
	}
	/*if err := prf.AccessClientsConfigure.Addresses[prf.ExternalIP].WsConnection.WriteMessage(1, formatJSON); err != nil {
		_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
	}*/

	fmt.Println("WEBSOCKET ERROR: ", err)

	fmt.Println("***************** STOP requestFilteringStart ************")

	fmt.Println("directory filtering: ", prf.AccessClientsConfigure.Addresses[prf.RemoteIP].TaskFilter[mft.Info.TaskIndex].DirectoryFiltering)
	fmt.Println("path storage filtering: ", prf.PathStorageFilterFiles)

	done := make(chan string, infoTaskFilter.CountDirectoryFiltering)

	listFilesFilter := prf.AccessClientsConfigure.Addresses[prf.RemoteIP].TaskFilter[mft.Info.TaskIndex].ListFilesFilter
	pathDirectoryFiltering := prf.AccessClientsConfigure.Addresses[prf.RemoteIP].TaskFilter[mft.Info.TaskIndex].DirectoryFiltering

	for dir := range listFilesFilter {
		patternParametersFiltering := PatternParametersFiltering{
			mft,
			dir,
			prf.TypeAreaNetwork,
			pathDirectoryFiltering,
			prf.AccessClientsConfigure.Addresses[prf.RemoteIP].TaskFilter[mft.Info.TaskIndex].ListFilesFilter[dir],
		}

		go filterProcessing(done, &patternParametersFiltering, patternBashScript(&patternParametersFiltering), prf)
	}

	formingMessageFilterComplete := FormingMessageFilterComplete{
		TaskIndex:      mft.Info.TaskIndex,
		RemoteIP:       prf.RemoteIP,
		CountDirectory: len(listFilesFilter),
		Done:           done,
	}

	//отправляем сообщение о завершении фильтрации
	go filteringComplete(&formingMessageFilterComplete, prf)

}

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

	//очищаем списки файлов по которым выполняется фильтрация
	for dirName := range prf.AccessClientsConfigure.Addresses[prf.ExternalIP].TaskFilter[mft.Info.TaskIndex].ListFilesFilter {
		prf.AccessClientsConfigure.Addresses[prf.ExternalIP].TaskFilter[mft.Info.TaskIndex].ListFilesFilter[dirName] = []string{}
	}

	var messageFilteringStop configure.MessageTypeFilteringStop
	messageFilteringStop.MessageType = "filtering"
	messageFilteringStop.Info.Processing = "stop"
	messageFilteringStop.Info.TaskIndex = mft.Info.TaskIndex
	messageFilteringStop.Info.IPAddress = prf.ExternalIP

	formatJSON, err := json.Marshal(&messageFilteringStop)
	if err != nil {
		_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
	}

	if err := prf.AccessClientsConfigure.Addresses[prf.RemoteIP].SendWsMessage(1, formatJSON); err != nil {
		_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
	}
	/*if err := prf.AccessClientsConfigure.Addresses[prf.ExternalIP].WsConnection.WriteMessage(1, formatJSON); err != nil {
		_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
	}*/

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
