package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

/*func checkFileName(dirName string, listFiles []string, pattern *regexp.Regexp, done chan<- struct{}, answer chan<- CheckedFile) {
	var checkedFile CheckedFile

	for _, file := range listFiles {
		if pattern.MatchString(file) {
			checkedFile.File = file
			checkedFile.Path = dirName

			answer <- checkedFile
		}
	}

	done <- struct{}{}
}

func checkNameFilesForFiltering(listFilesFilter map[string][]string, currentDisks []string) (map[string][]string, bool) {
	patterCheckFileName := regexp.MustCompile(regexpPatterns["fileName"])

	newListFilesFilter := map[string][]string{}
	done := make(chan struct{})
	answer := make(chan CheckedFile, len(currentDisks))
	defer func() {
		close(done)
		close(answer)
	}()

	for _, currentDisk := range currentDisks {
		for dir := range listFilesFilter {
			if currentDisk == dir {
				newListFilesFilter[currentDisk] = []string{}
				list := listFilesFilter[currentDisk]
				go checkFileName(currentDisk, list, patterCheckFileName, done, answer)
			}
		}
	}

	count := len(newListFilesFilter)
	for count != 0 {
		select {
		case <-done:
			count--
		case <-answer:
			result := <-answer
			newListFilesFilter[result.Path] = append(newListFilesFilter[result.Path], result.File)
		}
	}

	return newListFilesFilter, true
}*/

type CurrentListFilesFiltering struct {
	Path      string
	Files     []string
	SizeFiles int64
	ErrMsg    error
}

func getDateTimeRange(result chan<- CurrentListFilesFiltering, path string, dts, dte uint64) {
	var currentListFilesFiltering CurrentListFilesFiltering
	currentListFilesFiltering.Path = path
	currentListFilesFiltering.SizeFiles = 0

	fmt.Println("Search files for " + path + " directory")

	files, err := ioutil.ReadDir(path)
	if err != nil {
		currentListFilesFiltering.ErrMsg = err
		result <- currentListFilesFiltering
		return
	}

	//currentListFilesFiltering.List[path] = []string{}

	for _, file := range files {
		fileIsUnixDate := file.ModTime().Unix()
		if dts < uint64(fileIsUnixDate) && uint64(fileIsUnixDate) < dte {
			currentListFilesFiltering.Files = append(currentListFilesFiltering.Files, file.Name())
			currentListFilesFiltering.SizeFiles += file.Size()
		}
	}

	result <- currentListFilesFiltering
}

func getFilesList() (int, int64) {
	fmt.Println("---------------- function getFilesList is START -------------------")

	currenDisks := []string{
		"/__CURRENT_DISK_1",
		"/__CURRENT_DISK_2",
		"/__CURRENT_DISK_3",
		"/__CURRENT_DISK_21",
	}

	var dateTimeStart uint64 = 1461929460
	var dataTimeEnd uint64 = 1476447300

	listCountFiles := map[string]int{}

	var result = make(chan CurrentListFilesFiltering, len(currenDisks))

	for _, disk := range currenDisks {
		go getDateTimeRange(result, disk, dateTimeStart, dataTimeEnd)
	}

	var countFilesSearched int
	var sizeFilesSearched int64

	count := len(currenDisks)
	for count > 0 {
		resultFoundFile := <-result

		if resultFoundFile.ErrMsg != nil {

			fmt.Println("Error: ", resultFoundFile.ErrMsg)

		}

		listCountFiles[resultFoundFile.Path] = len(resultFoundFile.Files)
		countFilesSearched += len(resultFoundFile.Files)
		sizeFilesSearched += resultFoundFile.SizeFiles
		count--
	}
	close(result)

	fmt.Println("---------------- function getFilesList is END -------------------")
	fmt.Println(listCountFiles)

	return countFilesSearched, sizeFilesSearched
}

/*
func filesList(){
	done := make(chan struct{})


}
*/

func main() {

	//	n := math.Floor(100 / 30)
	//n := 100 / 20

	fmt.Println(float64(100) / 30)

	//fmt.Println(math.Trunc(n))

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

		fmt.Println("New network: ", newNetwork+"/"+networkTmp[1])

		return newNetwork, nil
	}

	res, err := getPatternNetwork("152.89.78.6/24")
	if err != nil {
		fmt.Println("Error get PatternNetwork: ", err)
	}
	fmt.Println("Error result: ", res)

	pattern := regexp.MustCompile(`^(\w|_)+\.(tdp|pcap)$`)
	testString := "1438528975_2015_08_02____18_22_55_1738.tdp"

	fmt.Println("File is True: ", pattern.MatchString(testString))

	//	countFiles, sizeFiles := getFilesList()
	//	fmt.Println("Files found = ", countFiles)
	//	fmt.Println("Files full size = ", sizeFiles)

	dateTimeStart := time.Unix(1461929460, 0)

	/*	newList := strings.FieldsFunc(dateTimeStart, func(c rune) bool {
		symbol := strconv.QuoteRune(c)
		return (symbol == " ")
	})*/
	fmt.Printf("%T %v", dateTimeStart, dateTimeStart.Year())
	go func() {

		fmt.Println("Groution START...")

		err := exec.Command("sh", "-c", "tcpdump -r /__CURRENT_DISK_1/1438535807_2015_08_02____20_16_47_753.tdp host 8.8.8.8 -w /home/TRAFFIC_FILTERS/2016_April_29_14_31_dsod999fvv/1438535807_2015_08_02____20_16_47_753.tdp").Run()
		if err != nil {
			fmt.Println(err)
		}
	}()

}
