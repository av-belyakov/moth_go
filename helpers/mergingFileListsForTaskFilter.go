package helpers

import (
	"errors"
	"fmt"

	"moth_go/configure"
)

/*
MergingFileListForTaskFilter выполняет объединение присылаемых клиентом списков
файлов необходимых для выполнения фильтрации (данной действие выполняется для индексных списков
или при возобновлении задачи по фильтрации)
*/
func MergingFileListForTaskFilter(ift *configure.InformationFilteringTask, mtf *configure.MessageTypeFilter) (error, bool) {
	if !mtf.Info.Settings.UseIndexes {
		return errors.New("task filtering not index"), true
	}

	fmt.Println("START function MergingFileListForTaskFilter")
	/*fmt.Println(mtf.Info.Settings.CountPartsIndexFiles)

	for dir, list := range mtf.Info.Settings.ListFilesFilter {
		fmt.Println(dir, "___________________", len(list))
	}*/

	//fmt.Println("NumberPleasantMessages == CountPartsIndexFiles[1]", ift.TaskID[mtf.Info.TaskIndex].NumberPleasantMessages == mtf.Info.Settings.CountPartsIndexFiles[1], " ", ift.TaskID[mtf.Info.TaskIndex].NumberPleasantMessages)

	if mtf.Info.Settings.CountPartsIndexFiles[0] == 0 {
		ift.TaskID[mtf.Info.TaskIndex].TotalNumberFilesFilter = mtf.Info.Settings.TotalNumberFilesFilter
		ift.TaskID[mtf.Info.TaskIndex].UseIndexes = true

		ift.TaskID[mtf.Info.TaskIndex].NumberPleasantMessages++

		fmt.Println("!!!!!! FERST ELEMENT")
		fmt.Println(ift.TaskID[mtf.Info.TaskIndex].TotalNumberFilesFilter)

		return nil, false
	}

	var countFiles, fullCountFiles int
	for dir, files := range mtf.Info.Settings.ListFilesFilter {
		/*
			fmt.Println(mtf.Info.Settings.CountPartsIndexFiles[0], " --- ", mtf.Info.Settings.CountPartsIndexFiles[1])
			fmt.Println("DIRECTORY NAME: ", dir)
			fmt.Println("COUNT FILES = ", len(files))
			fmt.Println("COUNT MERGIES FILES = ", len(ift.TaskID[mtf.Info.TaskIndex].ListFilesFilter[dir]))
		*/
		ift.TaskID[mtf.Info.TaskIndex].ListFilesFilter[dir] = append(ift.TaskID[mtf.Info.TaskIndex].ListFilesFilter[dir], files...)

		countFiles += len(files)
		fullCountFiles += len(ift.TaskID[mtf.Info.TaskIndex].ListFilesFilter[dir])
	}
	/*
		fmt.Println("mergingFileListsForTaskFilter RESIVED = ", countFiles)
		fmt.Println("FULL count files =", fullCountFiles)
		for directory, filesList := range ift.TaskID[mtf.Info.TaskIndex].ListFilesFilter {
			fmt.Println("directory name === ", directory, "num files === ", len(filesList))
		}
	*/
	if mtf.Info.Settings.CountPartsIndexFiles[0] == mtf.Info.Settings.CountPartsIndexFiles[1] {
		fmt.Println("!!!!!! LAST ELEMENT")

		for folder, value := range ift.TaskID[mtf.Info.TaskIndex].ListFilesFilter {
			fmt.Println("+===+ FOLDER ", folder, " count files = ", len(value))
		}

		return nil, true
	}

	//ift.TaskID[mtf.Info.TaskIndex].NumberPleasantMessages++

	return nil, false
}

//MergingFilesListForTaskFilter выполняет объединение присылаемых клиентом списков
/*func MergingFilesListForTaskFilter(chanErr chan<- error, chanDone chan<- struct{}, ift *configure.InformationFilteringTask, countParts int, chanMtf <-chan *configure.MessageTypeFilter) {
	fmt.Println("function MergingFilesListForTaskFilter ////? count Parts =", countParts)

	for i := 0; i < countParts; i++ {
		fmt.Println(i)
		mtf := <-chanMtf
		fmt.Println("chaneel =", mtf)

		if !mtf.Info.Settings.UseIndexes {
			chanErr <- errors.New("task filtering not index")
		}

		if mtf.Info.Settings.CountPartsIndexFiles[0] == 0 {
			ift.TaskID[mtf.Info.TaskIndex].TotalNumberFilesFilter = mtf.Info.Settings.TotalNumberFilesFilter

			fmt.Println("!!!!!! FERST ELEMENT")
		}

		for dir, files := range mtf.Info.Settings.ListFilesFilter {

			fmt.Println("DIRECTORY NAME: ", dir)
			fmt.Println("COUNT FILES = ", len(files))
			fmt.Println("COUNT MERGIES FILES = ", len(ift.TaskID[mtf.Info.TaskIndex].ListFilesFilter[dir]))

			ift.TaskID[mtf.Info.TaskIndex].ListFilesFilter[dir] = append(ift.TaskID[mtf.Info.TaskIndex].ListFilesFilter[dir], files...)

			if mtf.Info.Settings.CountPartsIndexFiles[0] == mtf.Info.Settings.CountPartsIndexFiles[1] {
				fmt.Println("!!!!!! LAST ELEMENT")

				for folder, value := range ift.TaskID[mtf.Info.TaskIndex].ListFilesFilter {
					fmt.Println("+===+ FOLDER ", folder, " count files = ", len(value))
				}

				chanDone <- struct{}{}
			}
		}
	}
}*/
