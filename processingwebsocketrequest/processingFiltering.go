package processingwebsocketrequest

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
	"moth_go/errormessage"
	"moth_go/savemessageapp"
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
func getListFilesForFiltering(prf *configure.ParametrsFunctionRequestFilter, mft *configure.MessageTypeFilter, ift *configure.InformationFilteringTask) /*(int, int64)*/ {
	//инициализируем функцию конструктор для записи лог-файлов
	saveMessageApp := savemessageapp.New()

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

		if resultFoundFile.Files != nil {
			currentTask.ListFilesFilter[resultFoundFile.Path] = resultFoundFile.Files
		}

		fullCountFiles += resultFoundFile.CountFiles
		fullSizeFiles += resultFoundFile.SizeFiles
		count--
	}

	//общее количество фильтруемых файлов
	currentTask.CountFilesFiltering = fullCountFiles
	//общий размер фильтруемых файлов
	currentTask.CountMaxFilesSize = fullSizeFiles
}

//формируем шаблон для фильтрации
func patternBashScript(ppf PatternParametersFiltering, mtf *configure.MessageTypeFilter) string {
	//инициализируем функцию конструктор для записи лог-файлов
	saveMessageApp := savemessageapp.New()

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
		if len(ipaddreses) != 0 {
			if len(ipaddreses) == 1 {
				searchHosts = "'(host " + ipaddreses[0] + " || (vlan && host " + ipaddreses[0] + "))'"
			} else {
				var hosts string
				for key, ip := range ipaddreses {
					if key == 0 {
						hosts += "(host " + ip
					} else if key == (len(ipaddreses) - 1) {
						hosts += " || " + ip + ")"
					} else {
						hosts += " || " + ip
					}
				}

				searchHosts = "'" + hosts + " || (vlan && " + hosts + ")'"
			}
		}
		return searchHosts
	}

	getNetworksString := func(networks []string) (searchNetworks string) {
		if len(networks) != 0 {
			if len(networks) == 1 {
				networkPattern, err := getPatternNetwork(networks[0])
				if err != nil {
					_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
				}

				searchNetworks += "'(net " + networkPattern + " || (vlan && net " + networkPattern + "))'"
			} else {
				var network string
				for key, net := range networks {
					networkPattern, err := getPatternNetwork(net)
					if err != nil {
						_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
					}

					if key == 0 {
						network += "(net " + networkPattern
					} else if key == (len(networks) - 1) {
						network += " || " + networkPattern + ")"
					} else {
						network += " || " + networkPattern
					}
				}
				searchNetworks = "'" + network + " || (vlan && " + network + ")'"
			}
		}
		return searchNetworks
	}

	bind := " "
	//формируем строку для поиска хостов
	searchHosts := getIPAddressString(mtf.Info.Settings.IPAddress)

	//формируем строку для поиска сетей
	searchNetwork := getNetworksString(mtf.Info.Settings.Network)

	if len(mtf.Info.Settings.IPAddress) > 0 && len(mtf.Info.Settings.Network) > 0 {
		bind = " or "
	}

	listTypeArea := map[int]string{
		1: " ",
		2: " '(pppoes && ip)' and ",
		3: " '(vlan && pppoes && ip)' and ",
		4: " '(pppoes && vlan && ip)' and ",
	}

	pattern := " tcpdump -r " + ppf.DirectoryName + "/$files"
	pattern += listTypeArea[ppf.TypeAreaNetwork] + searchHosts + bind + searchNetwork
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
		executeFiltering(prf, mft, ift)
	} else {
		if mft.Info.Settings.CountPartsIndexFiles[0] > 0 {
			if mft.Info.Settings.CountPartsIndexFiles[0] == mft.Info.Settings.CountPartsIndexFiles[1] {
				executeFiltering(prf, mft, ift)
			}
		}
	}
}

func executeFiltering(prf *configure.ParametrsFunctionRequestFilter, mtf *configure.MessageTypeFilter, ift *configure.InformationFilteringTask) {
	//инициализируем функцию конструктор для записи лог-файлов
	saveMessageApp := savemessageapp.New()

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
			responseDone = <-done
			if fmfc.taskIndex == responseDone.TaskIndex {
				dirComplete++
			}
		}

		//формируем канал для передачи информации о фильтрации
		prf.AccessClientsConfigure.ChanInfoFilterTask <- configure.ChanInfoFilterTask{
			TaskIndex:      fmfc.taskIndex,
			RemoteIP:       fmfc.remoteIP,
			TypeProcessing: responseDone.TypeProcessing,
		}

		//изменяем статус задачи на stop или complete
		ift.TaskID[taskIndex].TypeProcessing = responseDone.TypeProcessing

		defer close(done)
	}

	//список файлов для фильтрации
	getListFilesForFiltering(prf, mtf, ift)

	if ift.TaskID[taskIndex].CountMaxFilesSize == 0 {
		_ = saveMessageApp.LogMessage("info", "task ID "+taskIndex+", files needed to perform filtering not found")

		//сообщение пользователю
		if err := errormessage.SendErrorMessage(errormessage.Options{
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

		if err := errormessage.SendErrorMessage(errormessage.Options{
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
	//количество полных циклов
	infoTaskFilter.CountFullCycle = infoTaskFilter.CountFilesFiltering

	listCountFilesFilter := make(map[string]int)

	for disk := range ift.TaskID[taskIndex].ListFilesFilter {
		listCountFilesFilter[disk] = len(ift.TaskID[taskIndex].ListFilesFilter[disk])
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
				DirectoryFiltering:      infoTaskFilter.DirectoryFiltering,
				CountDirectoryFiltering: infoTaskFilter.CountDirectoryFiltering,
				CountFullCycle:          infoTaskFilter.CountFullCycle,
				CountFilesFiltering:     infoTaskFilter.CountFilesFiltering,
				CountMaxFilesSize:       infoTaskFilter.CountMaxFilesSize,
				UseIndexes:              mtf.Info.Settings.UseIndexes,
				NumberMessageParts:      numberMessageParts,
				ListCountFilesFilter:    listCountFilesFilter,
			},
		}

		formatJSON, err := json.Marshal(&messageFilteringStart)
		if err != nil {
			_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
		}

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
					UseIndexes:         mtf.Info.Settings.UseIndexes,
					NumberMessageParts: numberMessageParts,
					ListFilesFilter:    listFiles,
				},
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

	/*
	   ВНИМАНИЕ !!!
	   Отправку сообщения о начале фильтрации со списком фильтруемых файлов.
	   Список фильтруемых файлов в данном случае НЕ НУЖЕН, его надо убрать, так как он
	   только тормозит обработку а во Flashlight не используется.

	*/

	//отправка сообщения о начале фильтрации (первая часть без списка файлов)
	firstMessageStart()

	//продолжение отправки сообщений о начале фильтрации (со списком адресов)
	secondMessageStart(countPartsMessage)

	listFilesFilter := ift.TaskID[taskIndex].ListFilesFilter

	//ставим количество обработанных файлов
	ift.TaskID[taskIndex].CountFilesProcessed = mtf.Info.Settings.CountFilesFiltering - mtf.Info.Settings.TotalNumberFilesFilter

	done := make(chan ChanDone, infoTaskFilter.CountDirectoryFiltering)

	for dir := range listFilesFilter {
		if len(listFilesFilter[dir]) == 0 {
			continue
		}

		patternParametersFiltering := PatternParametersFiltering{
			TaskIndex:              taskIndex,
			DirectoryName:          dir,
			TypeAreaNetwork:        prf.TypeAreaNetwork,
			PathStorageFilterFiles: ift.TaskID[taskIndex].DirectoryFiltering,
			ListFiles:              &ift.TaskID[taskIndex].ListFilesFilter,
		}

		//запуск процесса фильтрации
		go filterProcessing(done, patternParametersFiltering, patternBashScript(patternParametersFiltering, mtf), prf, ift)
	}

	//изменяем статус задачи на execute
	ift.TaskID[taskIndex].TypeProcessing = "execute"

	formingMessageFilterComplete := FormingMessageFilterComplete{
		taskIndex:      taskIndex,
		remoteIP:       prf.RemoteIP,
		countDirectory: infoTaskFilter.CountDirectoryFiltering,
	}

	_ = saveMessageApp.LogMessage("info", "start of a task to filter with the ID "+mtf.Info.TaskIndex)

	//обработка завершения фильтрации
	go filteringComplete(done, &formingMessageFilterComplete, prf, ift)
}

//выполнение фильтрации
func filterProcessing(done chan<- ChanDone, ppf PatternParametersFiltering, patternBashScript string, prf *configure.ParametrsFunctionRequestFilter, ift *configure.InformationFilteringTask) {
	var statusProcessedFile bool

	//инициализируем функцию конструктор для записи лог-файлов
	saveMessageApp := savemessageapp.New()

	for _, file := range ift.TaskID[ppf.TaskIndex].ListFilesFilter[ppf.DirectoryName] {
		if _, ok := ift.TaskID[ppf.TaskIndex]; !ok {
			return
		}

		if ift.TaskID[ppf.TaskIndex].IsProcessStop {
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

	done <- ChanDone{
		TaskIndex:      ppf.TaskIndex,
		DirectoryName:  ppf.DirectoryName,
		TypeProcessing: "complete",
	}
}

func requestFilteringStop(prf *configure.ParametrsFunctionRequestFilter, mtf *configure.MessageTypeFilter, ift *configure.InformationFilteringTask) {
	fmt.Println("FILTER STOP, function requestFilteringStop START...")

	//инициализируем функцию конструктор для записи лог-файлов
	saveMessageApp := savemessageapp.New()

	taskIndex := mtf.Info.TaskIndex

	//проверка наличия задачи с указанным индексом
	if !ift.HasTaskFiltering(mtf.Info.TaskIndex) {
		if err := errormessage.SendErrorMessage(errormessage.Options{
			RemoteIP:   prf.RemoteIP,
			ErrMsg:     "no coincidenceId",
			TaskIndex:  taskIndex,
			ExternalIP: prf.ExternalIP,
			Wsc:        prf.AccessClientsConfigure.Addresses[prf.RemoteIP].WsConnection,
		}); err != nil {
			_ = saveMessageApp.LogMessage("error", "task ID "+taskIndex+" is not found")
		}

		return
	}

	ift.TaskID[taskIndex].IsProcessStop = true
}

//ProcessingFiltering обрабатывает запросы связанные с фильтрацией
func ProcessingFiltering(prf *configure.ParametrsFunctionRequestFilter, mtf *configure.MessageTypeFilter, ift *configure.InformationFilteringTask) {
	switch mtf.Info.Processing {
	case "on":
		requestFilteringStart(prf, mtf, ift)
	case "off":
		requestFilteringStop(prf, mtf, ift)
	}
}
