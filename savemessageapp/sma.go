package savemessageapp

import (
	"bufio"
	"compress/gzip"
	"errors"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"
)

func createLogsDirectory(directoryName string) error {
	files, err := ioutil.ReadDir(".")
	if err != nil {
		return err
	}

	for _, fl := range files {
		if fl.Name() == directoryName {
			return nil
		}
	}

	err = os.Mkdir(directoryName, 0777)
	if err != nil {
		return err
	}

	return nil
}

func compressLogFile(filePath string, fileName string, fileSize int64) error {
	fd, err := os.Open(filePath + fileName)
	if err != nil {
		return err
	}
	defer fd.Close()

	fileInfo, err := fd.Stat()
	if err != nil {
		return err
	}

	if fileInfo.Size() < fileSize {
		return nil
	}

	timeNowUnix := time.Now().Unix()
	newFileName := strconv.FormatInt(timeNowUnix, 10) + "_" + strings.Replace(fileName, ".log", ".gz", -1)

	fileIn, err := os.Create(filePath + newFileName)
	if err != nil {
		return err
	}
	defer fileIn.Close()

	zw := gzip.NewWriter(fileIn)
	zw.Name = newFileName

	fileOut, err := ioutil.ReadFile(filePath + fileName)
	if err != nil {
		return err
	}

	if _, err := zw.Write(fileOut); err != nil {
		return err
	}

	if err := zw.Close(); err != nil {
		return err
	}

	_ = os.Remove((filePath + fileName))

	return nil
}

//LogMessage сохраняет в лог файлах сообщения об ошибках или информационные сообщения
func LogMessage(typeMessage, message string) (err error) {
	const logDirName = "logs"
	const logFileSize = 100000000

	fileNameTypeMessage := map[string]string{
		"error": "error_message.log",
		"info":  "info_message.log",
	}

	if typeMessage == "" || message == "" {
		return errors.New("'typeMessage' or 'message' empty variable")
	}

	if err = createLogsDirectory(logDirName); err != nil {
		return err
	}

	_ = compressLogFile("./"+logDirName+"/", fileNameTypeMessage[typeMessage], logFileSize)

	var fileOut *os.File
	fileOut, err = os.OpenFile("./"+logDirName+"/"+fileNameTypeMessage[typeMessage], os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return err
	}

	timeNowString := time.Now().String()

	writer := bufio.NewWriter(fileOut)
	defer func() {
		if err == nil {
			err = writer.Flush()
		}
	}()

	if _, err = writer.WriteString(timeNowString + "\t" + message + "\n"); err != nil {
		return err
	}

	return err
}
