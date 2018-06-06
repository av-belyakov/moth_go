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
	fmt.Println(mtf.Info.Settings.CountPartsIndexFiles)

	for dir, list := range mtf.Info.Settings.ListFilesFilter {
		fmt.Println(dir, "___________________", len(list))
	}

	if mtf.Info.Settings.CountPartsIndexFiles[0] == 0 {
		ift.TaskID[mtf.Info.TaskIndex].TotalNumberFilesFilter = mtf.Info.Settings.TotalNumberFilesFilter

		fmt.Println("!!!!!! FERST ELEMENT")

		return nil, false
	}

	var countFiles, fullCountFiles int
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

			return nil, true
		}

		countFiles += len(files)
		fullCountFiles += len(ift.TaskID[mtf.Info.TaskIndex].ListFilesFilter[dir])
	}

	fmt.Println("mergingFileListsForTaskFilter RESIVED = ", countFiles)
	fmt.Println("FULL count files =", fullCountFiles)
	for directory, filesList := range ift.TaskID[mtf.Info.TaskIndex].ListFilesFilter {
		fmt.Println("directory name === ", directory, "num files === ", len(filesList))
	}

	return nil, false
}

//MergingFilesListForTaskFilter выполняет объединение присылаемых клиентом списков
/*func MergingFilesListForTaskFilter(chanErr chan<- error, chanDone chan<- bool, ift *configure.InformationFilteringTask, chanMtf <-chan *configure.MessageTypeFilter) {
	for {

	}

	if !mtf.Info.Settings.UseIndexes {
		chanErr <- errors.New("task filtering not index")
		chanDone <- true
		return
	}

	fmt.Println("START function MergingFileListForTaskFilter")
	fmt.Println(mtf.Info.Settings.CountPartsIndexFiles)

	for dir, list := range mtf.Info.Settings.ListFilesFilter {
		fmt.Println(dir, "___________________", len(list))
	}

	if mtf.Info.Settings.CountPartsIndexFiles[0] == 0 {
		ift.TaskID[mtf.Info.TaskIndex].TotalNumberFilesFilter = mtf.Info.Settings.TotalNumberFilesFilter

		fmt.Println("!!!!!! FERST ELEMENT")

		return nil, false
	}

	var countFiles, fullCountFiles int
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

			return nil, true
		}

		countFiles += len(files)
		fullCountFiles += len(ift.TaskID[mtf.Info.TaskIndex].ListFilesFilter[dir])
	}

	fmt.Println("mergingFileListsForTaskFilter RESIVED = ", countFiles)
	fmt.Println("FULL count files =", fullCountFiles)
	for directory, filesList := range ift.TaskID[mtf.Info.TaskIndex].ListFilesFilter {
		fmt.Println("directory name === ", directory, "num files === ", len(filesList))
	}

	return nil, false
}*/
