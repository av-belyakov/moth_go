package helpers

import (
	"fmt"
	"io/ioutil"
	"regexp"

	"moth_go/configure"
	"moth_go/saveMessageApp"
)

//CheckedFile содержит информацию о проверенном файле
type CheckedFile struct {
	Path, File string
}

var regexpPatterns = map[string]string{
	"IPAddress":                        `^((25[0-5]|2[0-4]\d|[01]?\d\d?)[.]){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$`,
	"Network":                          `^((25[0-5]|2[0-4]\d|[01]?\d\d?)[.]){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)/[0-9]{1,2}$`,
	"fileName":                         `^(\w|_)+\.(tdp|pcap)$`,
	"pathDirectoryStoryFilesFiltering": `^(\w|_|/)+$`,
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

func checkPathStorageFilterFiles(remoteIP string, mtdf configure.MessageTypeDownloadFiles, dfi *configure.DownloadFilesInformation) bool {
	fmt.Println("function checkPathStorageFilterFiles START...")

	patternCompile, err := regexp.Compile(regexpPatterns["pathDirectoryStoryFilesFiltering"])
	if err != nil {

		fmt.Println("CHECK ERROR 1")
		return false
	}

	ok := patternCompile.MatchString(mtdf.Info.DownloadDirectoryFiles)
	if !ok {

		fmt.Println("CHECK ERROR 2", mtdf.Info.DownloadDirectoryFiles)
		return false
	}

	dfi.RemoteIP[remoteIP].DirectoryFiltering = mtdf.Info.DownloadDirectoryFiles

	listFiles, err := ioutil.ReadDir(mtdf.Info.DownloadDirectoryFiles)
	if err != nil {

		fmt.Println("CHECK ERROR 3")
		return false
	}

	for _, files := range listFiles {
		dfi.RemoteIP[remoteIP].ListDownloadFiles[files.Name()] = &configure.FileInformationDownloadFiles{
			FileSize:               files.Size(),
			NumberTransferAttempts: 3,
		}
	}

	return true
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
func InputParametrsForFiltering(ift *configure.InformationFilteringTask, mtf *configure.MessageTypeFilter) (string, bool) {
	var ok bool

	//fmt.Printf("%v", mtf.Info.Settings)

	if (mtf.Info.Processing == "off") || (mtf.Info.Settings.UseIndexes && mtf.Info.Settings.CountPartsIndexFiles[0] > 0) {
		fmt.Println("++ +++ ++++ Message type is OFF or use INDEX and seconds chunk --- --- ---")

		return "", true
	}

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

	tID := mtf.Info.TaskIndex

	ift.TaskID[tID] = &configure.TaskInformation{
		FilterSettings: &configure.InfoFilterSettings{
			DateTimeStart: mtf.Info.Settings.DateTimeStart,
			DateTimeEnd:   mtf.Info.Settings.DateTimeEnd,
			IPAddress:     listIPAddress,
			Network:       listNetwork,
		},
		ListFilesFilter: map[string][]string{},
	}

	return "", true
}

//InputParametersForDownloadFile выполняется проверка валидности переданных пользователем данных
func InputParametersForDownloadFile(remoteIP string, mtdf configure.MessageTypeDownloadFiles, dfi *configure.DownloadFilesInformation) (string, bool) {
	//проверка наличия директории с которой выполняется выгрузка файлов
	if ok := checkPathStorageFilterFiles(remoteIP, mtdf, dfi); !ok {
		fmt.Println("CHECK PATH DIRECTORY FILTERING ERROR")

		_ = saveMessageApp.LogMessage("error", "task download files: incorrect received downloadDirectoryFiles")
		return "userDataIncorrect", false
	}

	return "", true
}
