package processingwebsocketrequest

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"

	"moth_go/configure"
	"moth_go/helpers"
	"moth_go/savemessageapp"
)

//SendMsgFilteringComplite отправляет сообщение о завершении фильтрации
func SendMsgFilteringComplite(acc *configure.AccessClientsConfigure, ift *configure.InformationFilteringTask, taskIndex string, task *configure.TaskInformation) {
	const sizeChunk = 30
	var messageTypeFilteringCompleteFirstPart configure.MessageTypeFilteringCompleteFirstPart

	//получить список найденных, в результате фильтрации, файлов
	getListFoundFiles := func(directoryResultFilter string) (list []configure.FoundFilesInfo, sizeFiles int64, err error) {
		files, err := ioutil.ReadDir(directoryResultFilter)
		if err != nil {
			return list, sizeFiles, err
		}

		var ffi configure.FoundFilesInfo
		for _, file := range files {
			if (file.Name() != "readme.txt") && (file.Size() > 24) {
				ffi.FileName = file.Name()
				ffi.FileSize = file.Size()

				sizeFiles += file.Size()

				list = append(list, ffi)
			}
		}

		return list, sizeFiles, nil
	}

	//получить количество частей сообщений
	getCountPartsMessage := func(list []configure.FoundFilesInfo, sizeChunk int) int {
		maxFiles := float64(len(list))

		newCountChunk := float64(sizeChunk)
		x := math.Floor(maxFiles / newCountChunk)
		y := maxFiles / newCountChunk

		if (y - x) != 0 {
			x++
		}

		return int(x)
	}

	listFoundFiles, sizeFiles, err := getListFoundFiles(task.DirectoryFiltering)
	if err != nil {
		_ = savemessageapp.LogMessage("error", fmt.Sprint(err))
	}

	messageTypeFilteringCompleteFirstPart = configure.MessageTypeFilteringCompleteFirstPart{
		MessageType: "filtering",
		Info: configure.MessageTypeFilteringCompleteInfoFirstPart{
			FilterInfoPattern: configure.FilterInfoPattern{
				Processing: task.TypeProcessing,
				TaskIndex:  taskIndex,
				IPAddress:  task.RemoteIP,
			},
			FilterCountPattern: configure.FilterCountPattern{
				CountFilesFound:       len(listFoundFiles),
				CountFoundFilesSize:   sizeFiles,
				CountCycleComplete:    task.CountCycleComplete,
				CountFilesProcessed:   task.CountFilesProcessed,
				CountFilesUnprocessed: task.CountFilesUnprocessed,
			},
		},
	}

	if len(listFoundFiles) == 0 {

		fmt.Println("--------------------- FILTERING COMPLETE (FILES NOT FOUND) -------------------")
		fmt.Println(messageTypeFilteringCompleteFirstPart)

		fmt.Println("++++++ job status: ", task.TypeProcessing, ", task ID:", taskIndex, "count files found:", task.CountFilesFound)

		formatJSON, err := json.Marshal(&messageTypeFilteringCompleteFirstPart)
		if err != nil {
			_ = savemessageapp.LogMessage("error", fmt.Sprint(err))
		}

		if _, ok := acc.Addresses[task.RemoteIP]; ok {
			acc.ChanWebsocketTranssmition <- formatJSON
		}
	} else {
		countPartsMessage := getCountPartsMessage(listFoundFiles, sizeChunk)

		numberMessageParts := [2]int{0, countPartsMessage}
		messageTypeFilteringCompleteFirstPart.Info.NumberMessageParts = numberMessageParts

		//отправляется первая часть сообщения
		formatJSON, err := json.Marshal(&messageTypeFilteringCompleteFirstPart)
		if err != nil {
			_ = savemessageapp.LogMessage("error", fmt.Sprint(err))
		}

		if _, ok := acc.Addresses[task.RemoteIP]; ok {
			acc.ChanWebsocketTranssmition <- formatJSON
		}

		//шаблон последующих частей сообщения
		var messageTypeFilteringCompleteSecondPart configure.MessageTypeFilteringCompleteSecondPart
		messageTypeFilteringCompleteSecondPart = configure.MessageTypeFilteringCompleteSecondPart{
			MessageType: "filtering",
			Info: configure.MessageTypeFilteringCompleteInfoSecondPart{
				FilterInfoPattern: configure.FilterInfoPattern{
					Processing: task.TypeProcessing,
					TaskIndex:  taskIndex,
					IPAddress:  task.RemoteIP,
				},
			},
		}

		//отправляются последующие части сообщений содержащие списки имен файлов
		for i := 1; i <= countPartsMessage; i++ {
			//получаем срез с частью имен файлов
			listFilesFoundDuringFiltering := helpers.GetChunkListFilesFoundDuringFiltering(configure.ChunkListParameters{
				NumPart:        i,
				CountParts:     countPartsMessage,
				SizeChunk:      sizeChunk,
				ListFoundFiles: listFoundFiles,
			})

			numberMessageParts[0] = i
			messageTypeFilteringCompleteSecondPart.Info.NumberMessageParts = numberMessageParts
			messageTypeFilteringCompleteSecondPart.Info.ListFilesFoundDuringFiltering = listFilesFoundDuringFiltering

			formatJSON, err := json.Marshal(&messageTypeFilteringCompleteSecondPart)
			if err != nil {
				_ = savemessageapp.LogMessage("error", fmt.Sprint(err))
			}

			if _, ok := acc.Addresses[task.RemoteIP]; ok {
				acc.ChanWebsocketTranssmition <- formatJSON
			}
		}
	}

	delete(ift.TaskID, taskIndex)
	_ = savemessageapp.LogMessage("info", task.TypeProcessing+" of the filter task execution with ID"+taskIndex)
}
