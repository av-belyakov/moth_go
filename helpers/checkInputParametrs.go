package helpers

import (
	"errors"
	"fmt"
	"io/ioutil"
	"regexp"
	"strings"

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

//CheckFileName проверяет имя файла на соответствие регулярному выражению
func CheckFileName(fileName, patternName string) error {
	pattern, found := regexpPatterns[patternName]
	if !found {
		return errors.New("function 'CheckFileName': not found the pattern for the regular expression")
	}

	fmt.Println("START function CheckFileName...")
	fmt.Println("fileName:", fileName, "patternName:", pattern)

	patterCheckFileName := regexp.MustCompile(pattern)
	if ok := patterCheckFileName.MatchString(fileName); ok {
		return nil
	}

	return errors.New("file name does not match the specified regular expression")
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

func checkFilesName(dirName string, listFiles []string, pattern *regexp.Regexp, done chan<- struct{}, answer chan<- CheckedFile) {
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

		fmt.Println("---- function checkPathStorageFilterFiles CHECK ERROR 1")
		return false
	}

	fmt.Println("---- function checkPathStorageFilterFiles CHECK 1 SUCCESS")

	ok := patternCompile.MatchString(mtdf.Info.DownloadDirectoryFiles)
	if !ok {

		fmt.Println("---- function checkPathStorageFilterFiles CHECK ERROR 2", mtdf.Info.DownloadDirectoryFiles)
		return false
	}

	fmt.Println("---- function checkPathStorageFilterFiles CHECK 2 SUCCESS")

	//если это не первое сообщение типа start (при передачи списка файлов необходимых дляскачивания)
	if mtdf.Info.NumberMessageParts[0] > 0 {
		return true
	}

	dfi.RemoteIP[remoteIP] = &configure.TaskInformationDownloadFiles{
		TaskIndex:               mtdf.Info.TaskIndex,
		DirectoryFiltering:      mtdf.Info.DownloadDirectoryFiles,
		SelectedFiles:           mtdf.Info.DownloadSelectedFiles,
		TotalCountDownloadFiles: mtdf.Info.CountDownloadSelectedFiles,
		ListDownloadFiles:       map[string]*configure.FileInformationDownloadFiles{},
	}

	if !mtdf.Info.DownloadSelectedFiles {
		listFiles, err := ioutil.ReadDir(mtdf.Info.DownloadDirectoryFiles)
		if err != nil {

			fmt.Println("---- function checkPathStorageFilterFiles CHECK ERROR 3")
			return false
		}

		fmt.Println("---- function checkPathStorageFilterFiles CHECK 3 SUCCESS")
		fmt.Println("-*-*-*-*-", dfi.RemoteIP[remoteIP], dfi.RemoteIP[remoteIP].ListDownloadFiles, "-*-*-*-*-")

		for _, f := range listFiles {
			if (f.Size() > 24) && (!strings.Contains(f.Name(), ".txt")) {
				dfi.RemoteIP[remoteIP].ListDownloadFiles[f.Name()] = &configure.FileInformationDownloadFiles{
					NumberTransferAttempts: 3,
				}
			}
		}

		fmt.Println("---- function checkPathStorageFilterFiles CREATE LIST FILES")

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
				go checkFilesName(currentDisk, list, patterCheckFileName, done, answer)
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
