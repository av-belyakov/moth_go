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

	fmt.Println("___________________", mtf.Info.Settings.CountPartsIndexFiles[0])

	if mtf.Info.Settings.CountPartsIndexFiles[0] == 0 {
		ift.TaskID[mtf.Info.TaskIndex].TotalNumberFilesFilter = mtf.Info.Settings.TotalNumberFilesFilter

		fmt.Println("!!!!!! FERST ELEMENT")

		return nil, false
	}

	for dir, files := range mtf.Info.Settings.ListFilesFilter {
		ift.TaskID[mtf.Info.TaskIndex].ListFilesFilter[dir] = append(ift.TaskID[mtf.Info.TaskIndex].ListFilesFilter[dir], files...)
	}

	for folder, value := range ift.TaskID[mtf.Info.TaskIndex].ListFilesFilter {
		fmt.Println("+===+ FOLDER ", folder, " count files = ", len(value))
	}

	if mtf.Info.Settings.CountPartsIndexFiles[0] == mtf.Info.Settings.CountPartsIndexFiles[1] {
		fmt.Println("!!!!!! LAST ELEMENT")

		return nil, true
	}

	return nil, false
}
