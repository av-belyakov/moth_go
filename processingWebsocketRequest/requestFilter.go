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
	TaskIndex              string
	DirectoryName          string
	TypeAreaNetwork        int
	PathStorageFilterFiles string
	ListFiles              *configure.ListFilesFilter
}

//FormingMessageFilterComplete содержит детали для отправки сообщения о завершении фильтрации
type FormingMessageFilterComplete struct {
	taskIndex      string
	remoteIP       string
	countDirectory int
}

//ChanDone содержит информацию о завершенной задаче
type ChanDone struct {
	TaskIndex, DirectoryName, TypeProcessing string
}

func searchFiles(result chan<- CurrentListFilesFiltering, disk string, currentTask *configure.TaskInformation) {
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
			if (currentTask.FilterSettings.DateTimeStart < uint64(fileIsUnixDate)) && (uint64(fileIsUnixDate) < currentTask.FilterSettings.DateTimeEnd) {
				currentListFilesFiltering.Files = append(currentListFilesFiltering.Files, file.Name())
				currentListFilesFiltering.SizeFiles += file.Size()
				currentListFilesFiltering.CountFiles++
			}
		}
	}

	result <- currentListFilesFiltering
}

//подготовка списка файлов по которым будет выполнятся фильтрация
func getListFilesForFiltering(prf *configure.ParametrsFunctionRequestFilter, mft *configure.MessageTypeFilter, ift *configure.InformationFilteringTask) (int, int64) {

	fmt.Println("START function getListFilesForFiltering...")

	currentTask := ift.TaskID[mft.Info.TaskIndex]

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
func patternBashScript(ppf PatternParametersFiltering, mtf *configure.MessageTypeFilter) string {
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
				searchHosts += " (host " + ipaddreses[0] + " || (vlan && host " + ipaddreses[0] + "))"
			} else {
				for _, ip := range ipaddreses {
					searchHosts += " (host " + ip + " || (vlan && host " + ip + "))"
					if num < (len(ipaddreses) - 1) {
						searchHosts += " ||"
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

				searchNetworks += " (net " + networkPattern + " || (vlan && net " + networkPattern + "))"
			} else {
				for _, net := range networks {
					networkPattern, err := getPatternNetwork(net)
					if err != nil {
						_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
					}

					searchNetworks += " (net " + networkPattern + " || (vlan && net " + networkPattern + "))"
					if num < (len(networks) - 1) {
						searchNetworks += " ||"
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
	searchHosts := getIPAddressString(mtf.Info.Settings.IPAddress)

	//формируем строку для поиска сетей
	searchNetwork := getNetworksString(mtf.Info.Settings.Network)

	if len(mtf.Info.Settings.IPAddress) > 0 && len(mtf.Info.Settings.Network) > 0 {
		bind = " || "
	}

	listTypeArea := map[int]string{
		1: "",
		2: " '(pppoes && ip)' and ",
	}

	pattern := " tcpdump -r " + ppf.DirectoryName + "/$files"
	pattern += listTypeArea[ppf.TypeAreaNetwork] + " '" + searchHosts + bind + searchNetwork + "'"
	pattern += " -w " + ppf.PathStorageFilterFiles + "/$files;"

	return pattern
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

func createDirectoryForFiltering(prf *configure.ParametrsFunctionRequestFilter, mft *configure.MessageTypeFilter, ift *configure.InformationFilteringTask) error {
	dateTimeStart := time.Unix(int64(mft.Info.Settings.DateTimeStart), 0)

	dirName := strconv.Itoa(dateTimeStart.Year()) + "_" + dateTimeStart.Month().String() + "_" + strconv.Itoa(dateTimeStart.Day()) + "_" + strconv.Itoa(dateTimeStart.Hour()) + "_" + strconv.Itoa(dateTimeStart.Minute()) + "_" + mft.Info.TaskIndex

	filePath := path.Join(prf.PathStorageFilterFiles, "/", dirName)

	ift.TaskID[mft.Info.TaskIndex].DirectoryFiltering = filePath

	return os.MkdirAll(filePath, 0766)
}

func requestFilteringStart(prf *configure.ParametrsFunctionRequestFilter, mft *configure.MessageTypeFilter, ift *configure.InformationFilteringTask) {
	//индексы не используются (в том числе и не возобновляется фильтрация)
	if !mft.Info.Settings.UseIndexes {
		fmt.Println("\nSTART filter not INDEX 1111")

		executeFiltering(prf, mft, ift)
	}

	if mft.Info.Settings.CountIndexesFiles[0] > 0 {

		fmt.Println("\nADD INDEX FILES IN ListFiles 2222")

		listFilesFilter := ift.TaskID[mft.Info.TaskIndex].ListFilesFilter

		for dir, listName := range mft.Info.Settings.ListFilesFilter {
			listFilesFilter[dir] = append(listFilesFilter[dir], listName...)
		}

		if mft.Info.Settings.CountIndexesFiles[0] == mft.Info.Settings.CountIndexesFiles[1] {
			fmt.Println("\nSTART FILTER WITH Index 3333")

			executeFiltering(prf, mft, ift)
		}
	}
}

func executeFiltering(prf *configure.ParametrsFunctionRequestFilter, mtf *configure.MessageTypeFilter, ift *configure.InformationFilteringTask) {
	fmt.Println("FILTER START, function requestFilteringStart START...")

	const sizeChunk = 30
	taskIndex := mtf.Info.TaskIndex

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
		directoryName := ift.TaskID[taskIndex].DirectoryFiltering

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
		text += "\tsearch date start: " + time.Unix(int64(mtf.Info.Settings.DateTimeStart), 0).String() + "\n"
		text += "\tsearch date end: " + time.Unix(int64(mtf.Info.Settings.DateTimeEnd), 0).String() + "\n"
		text += "\tsearch ipaddress: " + strings.Join(mtf.Info.Settings.IPAddress, "") + "\n"
		text += "\tsearch networks: " + strings.Join(mtf.Info.Settings.Network, "") + "\n"

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

	filteringComplete := func(done chan ChanDone, fmfc *FormingMessageFilterComplete, prf *configure.ParametrsFunctionRequestFilter, ift *configure.InformationFilteringTask) {
		var dirComplete int
		var responseDone ChanDone

		for dirComplete < fmfc.countDirectory {
			fmt.Println("dirComplete", dirComplete, "<", fmfc.countDirectory, "fmfc.countDirectory")

			//			responseDone = <-prf.AccessClientsConfigure.ChanCompleteDirTaskFilter
			responseDone = <-done
			if fmfc.taskIndex == responseDone.TaskIndex {
				fmt.Println("RESIVED STOP request, task ID", responseDone.TaskIndex)
				fmt.Println("dir name:", responseDone.DirectoryName, "\n")

				dirComplete++
			}
		}

		fmt.Println("==========================================")
		fmt.Println("--- FILTERING COMPLITE --- directory filtering is ", fmfc.countDirectory)
		fmt.Println("--- Task ID", fmfc.taskIndex)
		fmt.Println("==========================================")

		//формируем канал для передачи информации о фильтрации
		prf.AccessClientsConfigure.ChanInfoFilterTask <- configure.ChanInfoFilterTask{
			TaskIndex:      fmfc.taskIndex,
			RemoteIP:       fmfc.remoteIP,
			TypeProcessing: responseDone.TypeProcessing,
		}

		defer close(done)
	}

	//список файлов для фильтрации
	fullCountFiles, fullSizeFiles := getListFilesForFiltering(prf, mtf, ift)

	fmt.Println("full count files found to FILTER: ", fullCountFiles)
	fmt.Println("full size files found to FILTER: ", fullSizeFiles)

	if fullCountFiles == 0 {
		_ = saveMessageApp.LogMessage("info", "task ID: "+taskIndex+", files needed to perform filtering not found")

		//сообщение пользователю
		if err := errorMessage.SendErrorMessage(errorMessage.Options{
			RemoteIP:   prf.RemoteIP,
			ErrMsg:     "noFilesMatchingConfiguredInterval",
			TaskIndex:  taskIndex,
			ExternalIP: prf.ExternalIP,
			Wsc:        prf.AccessClientsConfigure.Addresses[prf.RemoteIP].WsConnection,
		}); err != nil {
			_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
		}

		//удаляем задачу из списка выполняемых задач
		delete(ift.TaskID, taskIndex)

		return
	}

	//создание директории для результатов фильтрации
	if err := createDirectoryForFiltering(prf, mtf, ift); err != nil {
		_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))

		if err := errorMessage.SendErrorMessage(errorMessage.Options{
			RemoteIP:   prf.RemoteIP,
			ErrMsg:     "serverError",
			TaskIndex:  taskIndex,
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

	infoTaskFilter := *ift.TaskID[taskIndex]
	//количество директорий для фильтрации
	infoTaskFilter.CountDirectoryFiltering = ift.GetCountDirectoryFiltering(taskIndex)
	//общее количество фильтруемых файлов
	infoTaskFilter.CountFilesFiltering = fullCountFiles
	//количество полных циклов
	infoTaskFilter.CountFullCycle = infoTaskFilter.CountFilesFiltering
	//общий размер фильтруемых файлов
	infoTaskFilter.CountMaxFilesSize = fullSizeFiles

	listCountFilesFilter := make(map[string]int)

	for disk := range ift.TaskID[taskIndex].ListFilesFilter {
		listCountFilesFilter[disk] = len(ift.TaskID[taskIndex].ListFilesFilter[disk])

		fmt.Println("DISK NAME: ", disk, " --- ", len(ift.TaskID[taskIndex].ListFilesFilter[disk]), " files")
	}

	countPartsMessage := getCountPartsMessage(listCountFilesFilter, sizeChunk)
	numberMessageParts := [2]int{0, countPartsMessage}

	firstMessageStart := func() {
		messageFilteringStart := configure.MessageTypeFilteringStartFirstPart{
			MessageType: "filtering",
			Info: configure.MessageTypeFilteringStartInfoFirstPart{
				FilterInfoPattern: configure.FilterInfoPattern{
					Processing: "start",
					TaskIndex:  taskIndex,
					IPAddress:  prf.ExternalIP,
				},
				FilterCountPattern: configure.FilterCountPattern{
					CountCycleComplete:    infoTaskFilter.CountCycleComplete,
					CountFilesFound:       infoTaskFilter.CountFilesFound,
					CountFoundFilesSize:   infoTaskFilter.CountFoundFilesSize,
					CountFilesProcessed:   infoTaskFilter.CountFilesProcessed,
					CountFilesUnprocessed: infoTaskFilter.CountFilesUnprocessed,
				},
				DirectoryFiltering:      infoTaskFilter.DirectoryFiltering,
				CountDirectoryFiltering: infoTaskFilter.CountDirectoryFiltering,
				CountFullCycle:          infoTaskFilter.CountFullCycle,
				CountFilesFiltering:     infoTaskFilter.CountFilesFiltering,
				CountMaxFilesSize:       infoTaskFilter.CountMaxFilesSize,
				UseIndexes:              false,
				NumberMessageParts:      numberMessageParts,
				ListCountFilesFilter:    listCountFilesFilter,
			},
		}

		formatJSON, err := json.Marshal(&messageFilteringStart)
		if err != nil {
			_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
		}

		fmt.Println("----- MESSAGE START FIRST -----")
		fmt.Println("-------------------------------------")
		fmt.Println("Count byte on the START format JSON message", len(formatJSON))
		fmt.Println("-------------------------------------")

		if err := prf.AccessClientsConfigure.Addresses[prf.RemoteIP].SendWsMessage(1, formatJSON); err != nil {
			_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
		}
	}

	secondMessageStart := func(countParts int) {
		getListFiles := func(numPart int) map[string][]string {
			listFilesFilter := map[string][]string{}

			for disk := range ift.TaskID[taskIndex].ListFilesFilter {

				lengthList := ift.TaskID[taskIndex].ListFilesFilter[disk]

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
				MessageType: "filtering",
				Info: configure.MessageTypeFilteringStartInfoSecondPart{
					FilterInfoPattern: configure.FilterInfoPattern{
						Processing: "start",
						TaskIndex:  taskIndex,
						IPAddress:  prf.ExternalIP,
					},
					NumberMessageParts: numberMessageParts,
					ListFilesFilter:    listFiles,
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

	listFilesFilter := ift.TaskID[taskIndex].ListFilesFilter

	fmt.Println("!!!!!!!!!!! COUNT DIRECTORY =", len(listFilesFilter))
	fmt.Println("count dir for filter (create CHAN DONE)", infoTaskFilter.CountDirectoryFiltering)

	done := make(chan ChanDone, infoTaskFilter.CountDirectoryFiltering)

	for dir := range listFilesFilter {
		patternParametersFiltering := PatternParametersFiltering{
			TaskIndex:              taskIndex,
			DirectoryName:          dir,
			TypeAreaNetwork:        prf.TypeAreaNetwork,
			PathStorageFilterFiles: ift.TaskID[taskIndex].DirectoryFiltering,
			ListFiles:              &ift.TaskID[taskIndex].ListFilesFilter,
		}

		fmt.Println("START process filter with task ID", taskIndex, "and directory name", dir)
		//		_ = saveMessageApp.LogMessage("info", "START process filter with task ID "+mtf.Info.TaskIndex+" and directory name "+dir)

		//запуск процесса фильтрации
		go filterProcessing(done, patternParametersFiltering, patternBashScript(patternParametersFiltering, mtf), prf, ift)
	}

	formingMessageFilterComplete := FormingMessageFilterComplete{
		taskIndex:      taskIndex,
		remoteIP:       prf.RemoteIP,
		countDirectory: infoTaskFilter.CountDirectoryFiltering,
	}

	_ = saveMessageApp.LogMessage("info", "the start of a task to filter with the ID"+mtf.Info.TaskIndex)

	//обработка завершения фильтрации
	go filteringComplete(done, &formingMessageFilterComplete, prf, ift)
}

//выполнение фильтрации
func filterProcessing(done chan<- ChanDone, ppf PatternParametersFiltering, patternBashScript string, prf *configure.ParametrsFunctionRequestFilter, ift *configure.InformationFilteringTask) {
	var statusProcessedFile bool

	fmt.Println(ppf.DirectoryName, " count files = ", len(ift.TaskID[ppf.TaskIndex].ListFilesFilter[ppf.DirectoryName]), "directory write", ppf.PathStorageFilterFiles)
	//	_ = saveMessageApp.LogMessage("info", " count files = "+string(len(ift.TaskID[ppf.TaskIndex].ListFilesFilter[ppf.DirectoryName]))+" directory write"+ppf.PathStorageFilterFiles)

	for _, file := range ift.TaskID[ppf.TaskIndex].ListFilesFilter[ppf.DirectoryName] {
		if _, ok := ift.TaskID[ppf.TaskIndex]; !ok {
			return
		}

		if ift.TaskID[ppf.TaskIndex].IsProcessStop {
			fmt.Println("************* IsProcessStop =", ift.TaskID[ppf.TaskIndex].IsProcessStop)
			fmt.Println("*************** task ID", ppf.TaskIndex)

			done <- ChanDone{
				TaskIndex:      ppf.TaskIndex,
				DirectoryName:  ppf.DirectoryName,
				TypeProcessing: "stop",
			}

			return
		}

		newPatternBashScript := strings.Replace(patternBashScript, "$files", file, -1)

		if err := exec.Command("sh", "-c", newPatternBashScript).Run(); err != nil {
			_ = saveMessageApp.LogMessage("error", fmt.Sprint(err)+"\t"+ppf.DirectoryName+", file: "+file)
			statusProcessedFile = false
		} else {
			statusProcessedFile = true
		}

		if _, ok := ift.TaskID[ppf.TaskIndex]; !ok {
			return
		}

		ift.TaskID[ppf.TaskIndex].CountCycleComplete++
		ift.TaskID[ppf.TaskIndex].CountFilesProcessed++

		if !statusProcessedFile {
			ift.TaskID[ppf.TaskIndex].CountFilesUnprocessed++
		}

		//получаем количество найденных файлов и их размер
		countFiles, fullSizeFiles, err := countNumberFilesFound(ppf.PathStorageFilterFiles)
		if err != nil {
			_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
		}

		//формируем канал для передачи информации о фильтрации
		prf.AccessClientsConfigure.ChanInfoFilterTask <- configure.ChanInfoFilterTask{
			TaskIndex:           ppf.TaskIndex,
			RemoteIP:            prf.RemoteIP,
			TypeProcessing:      "execute",
			DirectoryName:       ppf.DirectoryName,
			ProcessingFileName:  file,
			CountFilesFound:     countFiles,
			CountFoundFilesSize: fullSizeFiles,
			StatusProcessedFile: statusProcessedFile,
		}
	}

	fmt.Println("--------- Filter dir name:", ppf.DirectoryName, "STOP task ID", ppf.TaskIndex)

	done <- ChanDone{
		TaskIndex:      ppf.TaskIndex,
		DirectoryName:  ppf.DirectoryName,
		TypeProcessing: "complete",
	}
}

func requestFilteringStop(prf *configure.ParametrsFunctionRequestFilter, mtf *configure.MessageTypeFilter, ift *configure.InformationFilteringTask) {
	fmt.Println("FILTER STOP, function requestFilteringStop START...")

	taskIndex := mtf.Info.TaskIndex

	//проверка наличия задачи с указанным индексом
	if !ift.HasTaskFiltering(mtf.Info.TaskIndex) {
		if err := errorMessage.SendErrorMessage(errorMessage.Options{
			RemoteIP:   prf.RemoteIP,
			ErrMsg:     "no coincidenceId",
			TaskIndex:  taskIndex,
			ExternalIP: prf.ExternalIP,
			Wsc:        prf.AccessClientsConfigure.Addresses[prf.RemoteIP].WsConnection,
		}); err != nil {
			_ = saveMessageApp.LogMessage("error", "task ID"+taskIndex+"is not found")
		}

		return
	}

	ift.TaskID[taskIndex].IsProcessStop = true
}

//RequestTypeFilter обрабатывает запросы связанные с фильтрацией
func RequestTypeFilter(prf *configure.ParametrsFunctionRequestFilter, mtf *configure.MessageTypeFilter, ift *configure.InformationFilteringTask) {

	fmt.Println("\nFILTERING: function RequestTypeFilter STARTING...")
	fmt.Printf("%v", prf.AccessClientsConfigure)

	//проверяем количество одновременно выполняемых задач
	if ift.IsMaxConcurrentProcessFiltering(prf.RemoteIP, prf.AccessClientsConfigure.Addresses[prf.RemoteIP].MaxCountProcessFiltering) {
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
	if errMsg, ok := helpers.InputParametrsForFiltering(ift, mtf); !ok {
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

	switch mtf.Info.Processing {
	case "on":
		requestFilteringStart(prf, mtf, ift)
	case "off":
		requestFilteringStop(prf, mtf, ift)
	}
}
