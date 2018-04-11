package processingWebsocketRequest

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"net"
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
			if (currentTask.DateTimeStart < uint64(fileIsUnixDate)) && (uint64(fileIsUnixDate) < currentTask.DateTimeEnd) {
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
	getPatternNetwork := func(network string) (string, error) {
		networkTmp := strings.Split(network, "/")
		if len(networkTmp) < 2 {
			return "", errors.New("incorrect network mask value")
		}

		maskInt, err := strconv.ParseInt(networkTmp[1], 10, 64)
		if err != nil {
			return "", err
		}

		if maskInt < 0 || maskInt > 32 {
			return "", errors.New("the value of 'mask' should be in the range from 0 to 32")
		}

		ipv4Addr := net.ParseIP(networkTmp[0])
		ipv4Mask := net.CIDRMask(24, 32)
		newNetwork := ipv4Addr.Mask(ipv4Mask).String()

		return newNetwork + "/" + networkTmp[1], nil
	}

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
				networkPattern, err := getPatternNetwork(networks[0])
				if err != nil {
					_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
				}

				searchNetworks += " net " + networkPattern
			} else {
				for _, net := range networks {
					networkPattern, err := getPatternNetwork(net)
					if err != nil {
						_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
					}

					searchNetworks += " net " + networkPattern
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

	return pattern
}

//выполнение фильтрации
func filterProcessing(done chan<- string, patternParametersFiltering *PatternParametersFiltering, patternBashScript string, prf *configure.ParametrsFunctionRequestFilter) {

	fmt.Println("START func filterProcessing..........................", patternParametersFiltering.DirectoryName)

	listFilesFilter := patternParametersFiltering.ListFiles
	task := prf.AccessClientsConfigure.Addresses[prf.RemoteIP].TaskFilter[patternParametersFiltering.ParameterFilter.Info.TaskIndex]

	//формируем сообщение о ходе фильтрации
	var mtfeou configure.MessageTypeFilteringExecutedOrUnexecuted
	mtfeou.MessageType = "filtering"
	mtfeou.Info.ProcessingFile.DirectoryLocation = patternParametersFiltering.DirectoryName
	mtfeou.Info.IPAddress = prf.RemoteIP
	mtfeou.Info.TaskIndex = patternParametersFiltering.ParameterFilter.Info.TaskIndex

	fmt.Println(patternParametersFiltering.DirectoryName, " count files = ", len(listFilesFilter))

	for _, file := range listFilesFilter {
		mtfeou.Info.ProcessingFile.FileName = file

		newPatternBashScript := strings.Replace(patternBashScript, "$files", file, -1)

		task.CountCycleComplete++
		task.CountFilesProcessed++
		mtfeou.Info.CountFilesProcessed = task.CountFilesProcessed

		mtfeou.Info.Processing = "execute"
		if err := exec.Command("sh", "-c", newPatternBashScript).Run(); err != nil {
			_ = saveMessageApp.LogMessage("error", fmt.Sprint(err)+"\t"+patternParametersFiltering.DirectoryName+", file: "+file)

			task.CountFilesUnprocessed++
			mtfeou.Info.CountFilesUnprocessed = task.CountFilesUnprocessed
			mtfeou.Info.ProcessingFile.StatusProcessed = false
		} else {
			mtfeou.Info.ProcessingFile.StatusProcessed = true
		}

		//получаем количество найденных файлов и их размер
		countFiles, fullSizeFiles, err := countNumberFilesFound(patternParametersFiltering.PathStorageFilterFiles)
		if err != nil {
			_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
		}

		mtfeou.Info.CountCycleComplete = task.CountCycleComplete
		mtfeou.Info.CountFilesFound = countFiles
		mtfeou.Info.CountFoundFilesSize = fullSizeFiles

		formatJSON, err := json.Marshal(&mtfeou)
		if err != nil {
			_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
		}

		if err := prf.AccessClientsConfigure.Addresses[prf.RemoteIP].SendWsMessage(1, formatJSON); err != nil {
			_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
		}
	}
	done <- patternParametersFiltering.DirectoryName
}

//подстчет количества найденных файлов
func countNumberFilesFound(directoryResultFilter string) (count int, size int64, err error) {
	files, err := ioutil.ReadDir(directoryResultFilter)
	if err != nil {
		return count, size, err
	}

	for _, file := range files {
		if (file.Name() != "readme.txt") && (file.Size() > 24) {
			size += file.Size()
			count++
		}
	}

	if count > 0 {
		count--
	}

	return count, size, nil
}

func createDirectoryForFiltering(prf *configure.ParametrsFunctionRequestFilter, mft *configure.MessageTypeFilter) error {

	//dateTimeStart := time.Unix(1461929460, 0)
	dateTimeStart := time.Unix(int64(mft.Info.Settings.DateTimeStart), 0)

	dirName := strconv.Itoa(dateTimeStart.Year()) + "_" + dateTimeStart.Month().String() + "_" + strconv.Itoa(dateTimeStart.Day()) + "_" + strconv.Itoa(dateTimeStart.Hour()) + "_" + strconv.Itoa(dateTimeStart.Minute()) + "_" + mft.Info.TaskIndex

	filePath := path.Join(prf.PathStorageFilterFiles, "/", dirName)

	prf.AccessClientsConfigure.Addresses[prf.RemoteIP].TaskFilter[mft.Info.TaskIndex].DirectoryFiltering = filePath

	return os.MkdirAll(filePath, 0766)
}

func requestFilteringStart(prf *configure.ParametrsFunctionRequestFilter, mft *configure.MessageTypeFilter) {
	//индексы не используются (в том числе и не возобновляется фильтрация)
	if !mft.Info.Settings.UseIndexes {
		fmt.Println("\nSTART filter not INDEX 1111")

		executeFiltering(prf, mft)
	}

	if mft.Info.Settings.CountIndexesFiles[0] > 0 {
		fmt.Println("\nADD INDEX FILES IN ListFiles 2222")
		listFilesFilter := prf.AccessClientsConfigure.Addresses[prf.RemoteIP].TaskFilter[mft.Info.TaskIndex].ListFilesFilter

		for dir, listName := range mft.Info.Settings.ListFilesFilter {
			listFilesFilter[dir] = append(listFilesFilter[dir], listName...)
		}

		if mft.Info.Settings.CountIndexesFiles[0] == mft.Info.Settings.CountIndexesFiles[1] {
			fmt.Println("\nSTART FILTER WITH Index 3333")

			executeFiltering(prf, mft)
		}
	}
}

func executeFiltering(prf *configure.ParametrsFunctionRequestFilter, mft *configure.MessageTypeFilter) {
	fmt.Println("FILTER START, function requestFilteringStart START...")

	const sizeChunk = 30

	getCountPartsMessage := func(list map[string]int, sizeChunk int) int {
		var maxFiles float64
		for _, v := range list {
			if maxFiles < float64(v) {
				maxFiles = float64(v)
			}
		}

		newCountChunk := float64(sizeChunk)
		x := math.Floor(maxFiles / newCountChunk)
		y := maxFiles / newCountChunk

		if (y - x) != 0 {
			x++
		}

		return int(x)
	}

	createFileReadme := func() error {
		directoryName := prf.AccessClientsConfigure.Addresses[prf.RemoteIP].TaskFilter[mft.Info.TaskIndex].DirectoryFiltering

		files, err := ioutil.ReadDir(directoryName)
		if err != nil {
			return err
		}

		for _, file := range files {
			if file.Name() == "readme.txt" {
				return nil
			}
		}

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
		var dirComplete int
		var dirNameComplete string

		task := prf.AccessClientsConfigure.Addresses[prf.RemoteIP].TaskFilter[fmfc.TaskIndex]

		fmt.Println("START func filteringComplete ++++++")

		for dirComplete < fmfc.CountDirectory {
			dirNameComplete = <-fmfc.Done

			fmt.Println("DIRECTORY NAME = ", dirNameComplete)

			if len(dirNameComplete) > 0 {
				dirComplete++
			}
		}

		fmt.Println("==========================================")
		fmt.Println("--- FILTERING COMPLITE --- directory filtering is ", fmfc.CountDirectory)
		fmt.Println("==========================================")

		close(fmfc.Done)

		messageTypeFilteringComplete := configure.MessageTypeFilteringComplete{
			"filtering",
			configure.MessageTypeFilteringCompleteInfo{
				FilterinInfoPattern: configure.FilterinInfoPattern{
					Processing: "complete",
					TaskIndex:  fmfc.TaskIndex,
					IPAddress:  fmfc.RemoteIP,
				},
				CountCycleComplete: task.CountCycleComplete,
			},
		}

		formatJSON, err := json.Marshal(&messageTypeFilteringComplete)
		if err != nil {
			_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
		}

		if err := prf.AccessClientsConfigure.Addresses[prf.RemoteIP].SendWsMessage(1, formatJSON); err != nil {
			_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
		}

		//удаляем задачу
		delete(prf.AccessClientsConfigure.Addresses[fmfc.RemoteIP].TaskFilter, fmfc.TaskIndex)
	}

	//список файлов для фильтрации
	fullCountFiles, fullSizeFiles := getListFilesForFiltering(prf, mft)

	fmt.Println("full count files found to FILTER: ", fullCountFiles)
	fmt.Println("full size files found to FILTER: ", fullSizeFiles)

	if fullCountFiles == 0 {
		_ = saveMessageApp.LogMessage("info", "task ID: "+mft.Info.TaskIndex+", files needed to perform filtering not found")

		if err := errorMessage.SendErrorMessage(errorMessage.Options{
			RemoteIP:   prf.RemoteIP,
			ErrMsg:     "noFilesMatchingConfiguredInterval",
			TaskIndex:  mft.Info.TaskIndex,
			ExternalIP: prf.ExternalIP,
			Wsc:        prf.AccessClientsConfigure.Addresses[prf.RemoteIP].WsConnection,
		}); err != nil {
			_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
		}
		return
	}

	//создание директории для результатов фильтрации
	if err := createDirectoryForFiltering(prf, mft); err != nil {
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

	listCountFilesFilter := make(map[string]int)

	for disk := range prf.AccessClientsConfigure.Addresses[prf.RemoteIP].TaskFilter[mft.Info.TaskIndex].ListFilesFilter {
		listCountFilesFilter[disk] = len(prf.AccessClientsConfigure.Addresses[prf.RemoteIP].TaskFilter[mft.Info.TaskIndex].ListFilesFilter[disk])

		fmt.Println("DISK NAME: ", disk, " --- ", len(prf.AccessClientsConfigure.Addresses[prf.RemoteIP].TaskFilter[mft.Info.TaskIndex].ListFilesFilter[disk]), " files")
	}

	countPartsMessage := getCountPartsMessage(listCountFilesFilter, sizeChunk)
	numberMessageParts := [2]int{0, countPartsMessage}

	firstMessageStart := func() {
		messageFilteringStart := configure.MessageTypeFilteringStartFirstPart{
			"filtering",
			configure.MessageTypeFilteringStartInfoFirstPart{
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
				numberMessageParts,
				listCountFilesFilter,
			},
		}

		formatJSON, err := json.Marshal(&messageFilteringStart)
		if err != nil {
			_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
		}

		fmt.Println("----- MESSAGE START FIRST -----")
		fmt.Println("-------------------------------------")
		fmt.Println("Count byte on the START", len(formatJSON))
		fmt.Println("-------------------------------------")

		if err := prf.AccessClientsConfigure.Addresses[prf.RemoteIP].SendWsMessage(1, formatJSON); err != nil {
			_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
		}
	}

	secondMessageStart := func(countParts int) {

		getListFiles := func(numPart int) map[string][]string {
			listFilesFilter := map[string][]string{}

			for disk := range prf.AccessClientsConfigure.Addresses[prf.RemoteIP].TaskFilter[mft.Info.TaskIndex].ListFilesFilter {

				lengthList := prf.AccessClientsConfigure.Addresses[prf.RemoteIP].TaskFilter[mft.Info.TaskIndex].ListFilesFilter[disk]

				if numPart == 1 {
					if len(lengthList) < sizeChunk {
						listFilesFilter[disk] = lengthList[:]
					} else {
						listFilesFilter[disk] = lengthList[:sizeChunk]
					}
				} else {
					num := sizeChunk * (numPart - 1)
					numEnd := num + sizeChunk

					if (numPart == countParts) && (num < len(lengthList)) {
						listFilesFilter[disk] = lengthList[num:]
					}
					if (numPart < countParts) && (numEnd < len(lengthList)) {
						listFilesFilter[disk] = lengthList[num:numEnd]
					}

				}
			}
			return listFilesFilter
		}

		for i := 1; i <= countParts; i++ {
			listFiles := getListFiles(i)

			numberMessageParts[0] = i
			messageFilteringStart := configure.MessageTypeFilteringStartSecondPart{
				"filtering",
				configure.MessageTypeFilteringStartInfoSecondPart{
					configure.FilterinInfoPattern{
						Processing: "start",
						TaskIndex:  mft.Info.TaskIndex,
						IPAddress:  prf.ExternalIP,
					},
					numberMessageParts,
					listFiles,
				},
			}

			fmt.Println("----- MESSAGE START SECOND -----")
			for key, v := range listFiles {
				fmt.Println(key, " = ", len(v))
			}

			formatJSON, err := json.Marshal(&messageFilteringStart)
			if err != nil {
				_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
			}

			if err := prf.AccessClientsConfigure.Addresses[prf.RemoteIP].SendWsMessage(1, formatJSON); err != nil {
				_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
			}
		}
	}

	//отправка сообщения о начале фильтрации (первая часть без списка файлов)
	firstMessageStart()

	//продолжение отправки сообщений о начале фильтрации (со списком адресов)
	secondMessageStart(countPartsMessage)

	done := make(chan string, infoTaskFilter.CountDirectoryFiltering)

	listFilesFilter := prf.AccessClientsConfigure.Addresses[prf.RemoteIP].TaskFilter[mft.Info.TaskIndex].ListFilesFilter
	pathDirectoryFiltering := prf.AccessClientsConfigure.Addresses[prf.RemoteIP].TaskFilter[mft.Info.TaskIndex].DirectoryFiltering

	fmt.Println("!!!!!!!!!!! COUNT DIRECTORY =", len(listFilesFilter))
	fmt.Println("count dir for filter (create CHAN DONE)", infoTaskFilter.CountDirectoryFiltering)

	for dir := range listFilesFilter {
		patternParametersFiltering := PatternParametersFiltering{
			mft,
			dir,
			prf.TypeAreaNetwork,
			pathDirectoryFiltering,
			prf.AccessClientsConfigure.Addresses[prf.RemoteIP].TaskFilter[mft.Info.TaskIndex].ListFilesFilter[dir],
		}

		//запуск процесса фильтрации
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

	fmt.Println("FILTER STOP, function requestFilteringStop START...")

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

	delete(prf.AccessClientsConfigure.Addresses[prf.RemoteIP].TaskFilter, mft.Info.TaskIndex)
}

//RequestTypeFilter обрабатывает запросы связанные с фильтрацией
func RequestTypeFilter(prf *configure.ParametrsFunctionRequestFilter, mtf *configure.MessageTypeFilter) {

	fmt.Println("\nFILTERING: function RequestTypeFilter STARTING...")

	//проверяем количество одновременно выполняемых задач
	if prf.AccessClientsConfigure.IsMaxCountProcessFiltering(prf.RemoteIP) {
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

	//проверка полученных от пользователя данных (дата и время, список адресов и сетей)
	if errMsg, ok := helpers.InputParametrsForFiltering(prf, mtf); !ok {
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

	typeRequest := mtf.Info.Processing
	switch typeRequest {
	case "on":
		requestFilteringStart(prf, mtf)
	case "off":
		requestFilteringStop(prf, mtf)
	}
}
