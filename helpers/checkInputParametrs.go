package helpers

import (
	"fmt"
	"regexp"

	"moth_go/configure"
	"moth_go/saveMessageApp"
)

//CheckedFile содержит информацию о проверенном файле
type CheckedFile struct {
	Path, File string
}

var regexpPatterns = map[string]string{
	"IPAddress": `^((25[0-5]|2[0-4]\d|[01]?\d\d?)[.]){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$`,
	"Network":   `^((25[0-5]|2[0-4]\d|[01]?\d\d?)[.]){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)/[0-9]{1,2}$`,
	"fileName":  `^(\w|_)+\.(tdp|pcap)$`,
}

func checkDateTime(dts, dte uint64) bool {
	if dts == 0 || dte == 0 {
		return false
	}

	if dts > dte {
		return false
	}

	return true
}

func checkNetworkOrIPAddress(typeValue string, value []string) ([]string, bool) {
	var validValues []string
	pattern := regexpPatterns[typeValue]
	if pattern == "" {
		return validValues, false
	}

	patternCompile, err := regexp.Compile(pattern)
	if err != nil {
		return validValues, false
	}

	for _, v := range value {
		ok := patternCompile.MatchString(v)
		if ok {
			validValues = append(validValues, v)
		}
	}

	return validValues, true
}

func checkFileName(dirName string, listFiles []string, pattern *regexp.Regexp, done chan<- struct{}, answer chan<- CheckedFile) {
	var checkedFile CheckedFile

	for _, file := range listFiles {
		if ok := pattern.MatchString(file); ok {
			checkedFile.File = file
			checkedFile.Path = dirName

			answer <- checkedFile
		}
	}

	done <- struct{}{}
}

func awaitCompletion(done <-chan struct{}, countCycle int, answer chan CheckedFile) {
	for i := 0; i < countCycle; i++ {
		<-done
	}
	close(answer)
}

func checkNameFilesForFiltering(listFilesFilter map[string][]string, currentDisks []string) (map[string][]string, bool) {
	patterCheckFileName := regexp.MustCompile(regexpPatterns["fileName"])

	fmt.Println("function checkNameFilesForFiltering START...")

	newListFilesFilter := map[string][]string{}
	done := make(chan struct{}, cap(currentDisks))
	answer := make(chan CheckedFile, len(currentDisks))
	defer close(done)

	var count int
	for _, currentDisk := range currentDisks {
		for dir := range listFilesFilter {
			if currentDisk == dir {
				count++
				newListFilesFilter[currentDisk] = []string{}
				list := listFilesFilter[currentDisk]
				go checkFileName(currentDisk, list, patterCheckFileName, done, answer)
			}
		}
	}

	listFilesIsEmpty := true

	//закрываем канал 'answer' после завершения всех функций обработчиков
	go awaitCompletion(done, count, answer)

	for result := range answer {
		newListFilesFilter[result.Path] = append(newListFilesFilter[result.Path], result.File)
		listFilesIsEmpty = false
	}

	if listFilesIsEmpty {
		return newListFilesFilter, false
	}

	return newListFilesFilter, true
}

//InputParametrsForFiltering выполняет ряд проверок валидности полученных от пользователя данных перед выполнением фильтрации
func InputParametrsForFiltering(prf *configure.ParametrsFunctionRequestFilter, mtf *configure.MessageTypeFilter) (string, bool) {
	var ok bool
	if mtf.Info.Processing == "on" || mtf.Info.Processing == "resume" {
		//проверяем дату и время
		if ok = checkDateTime(mtf.Info.Settings.DateTimeStart, mtf.Info.Settings.DateTimeEnd); !ok {
			fmt.Println("CHECK DATETIME ERROR")

			_ = saveMessageApp.LogMessage("error", "task filtering: incorrect received datetime")
			return "userDataIncorrect", false
		}

		//проверяем IP адреса
		listIPAddress, ok := checkNetworkOrIPAddress("IPAddress", mtf.Info.Settings.IPAddress)
		if !ok {
			fmt.Println("CHECK IPADDRESS ERROR")

			_ = saveMessageApp.LogMessage("error", "task filtering: incorrect ipaddress")
			return "userDataIncorrect", false
		}

		//проверяем диапазоны сетей
		listNetwork, ok := checkNetworkOrIPAddress("Network", mtf.Info.Settings.Network)
		if !ok {
			fmt.Println("CHECK NETWORK ERROR")

			_ = saveMessageApp.LogMessage("error", "task filtering: incorrect received network")
			return "userDataIncorrect", false
		}

		//проверяем наличие списков ip адресов или подсетей
		if len(listIPAddress) == 0 && len(listNetwork) == 0 {
			_ = saveMessageApp.LogMessage("error", "task filtering: an empty list of addresses or networks found")
			return "userDataIncorrect", false
		}

		//проверяем пути и имена файлов которые необходимо отфильтровать
		listFilesFilter, ok := checkNameFilesForFiltering(mtf.Info.Settings.ListFilesFilter, prf.CurrentDisks)
		if mtf.Info.Settings.UseIndexes && !ok {
			fmt.Println("CHECK LIST_FILES_FILTERING ERROR")

			_ = saveMessageApp.LogMessage("error", "task filtering: incorrect received list files filtering")
			return "userDataIncorrect", false
		}

		tID := mtf.Info.TaskIndex
		acc := prf.AccessClientsConfigure.Addresses[prf.RemoteIP]

		var taskFilter = make(map[string]*configure.InformationTaskFilter)
		taskFilter[tID] = &configure.InformationTaskFilter{}
		acc.TaskFilter = taskFilter

		acc.TaskFilter[tID].DateTimeStart = mtf.Info.Settings.DateTimeStart
		acc.TaskFilter[tID].DateTimeEnd = mtf.Info.Settings.DateTimeEnd
		acc.TaskFilter[tID].IPAddress = listIPAddress
		acc.TaskFilter[tID].Network = listNetwork
		acc.TaskFilter[tID].UseIndexes = mtf.Info.Settings.UseIndexes
		acc.TaskFilter[tID].ListFilesFilter = listFilesFilter

		return "", true
	}
	return "", true
}
