package helpers

import (
	"errors"

	"moth_go/configure"
)

/*
MergingFileListForTaskFilter выполняет объединение присылаемых клиентом списков
файлов необходимых для выполнения фильтрации (данной действие выполняется для индексных списков
или при возобновлении задачи по фильтрации)
*/
func MergingFileListForTaskFilter(ift *configure.InformationFilteringTask, mtf *configure.MessageTypeFilter) (bool, error) {
	if !mtf.Info.Settings.UseIndexes {
		return true, errors.New("task filtering not index")
	}

	if mtf.Info.Settings.CountPartsIndexFiles[0] == 0 {
		ift.TaskID[mtf.Info.TaskIndex].TotalNumberFilesFilter = mtf.Info.Settings.TotalNumberFilesFilter
		ift.TaskID[mtf.Info.TaskIndex].UseIndexes = true

		ift.TaskID[mtf.Info.TaskIndex].NumberPleasantMessages++

		return false, nil
	}

	var countFiles, fullCountFiles int
	for dir, files := range mtf.Info.Settings.ListFilesFilter {
		ift.TaskID[mtf.Info.TaskIndex].ListFilesFilter[dir] = append(ift.TaskID[mtf.Info.TaskIndex].ListFilesFilter[dir], files...)

		countFiles += len(files)
		fullCountFiles += len(ift.TaskID[mtf.Info.TaskIndex].ListFilesFilter[dir])
	}

	if mtf.Info.Settings.CountPartsIndexFiles[0] == mtf.Info.Settings.CountPartsIndexFiles[1] {
		return true, nil
	}

	return false, nil
}
