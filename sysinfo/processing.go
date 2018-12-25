package sysinfo

import (
	"encoding/json"
	"fmt"
	"time"

	"moth_go/configure"
	"moth_go/savemessageapp"
)

//GetSystemInformation позволяет получить системную информацию
func GetSystemInformation(out chan<- []byte, mc *configure.MothConfig) {
	//инициализируем функцию конструктор для записи лог-файлов
	saveMessageApp := savemessageapp.New()

	var sysInfo configure.SysInfo

	chanErrMsg := make(chan error)
	done := make(chan struct{})

	//временной интервал файлов хранящихся на дисках
	go sysInfo.CreateFilesRange(done, chanErrMsg, mc.CurrentDisks)

	//нагрузка на сетевых интерфейсах
	go sysInfo.CreateLoadNetworkInterface(done, chanErrMsg)

	//загрузка оперативной памяти
	if err := sysInfo.CreateRandomAccessMemory(); err != nil {
		_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
	}

	//загрузка ЦПУ
	if err := sysInfo.CreateLoadCPU(); err != nil {
		_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
	}

	//свободное дисковое пространство
	if err := sysInfo.CreateDiskSpace(); err != nil {
		_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
	}

	numGoProg := 2

DONE:
	for {
		select {
		case errMsg := <-chanErrMsg:
			_ = saveMessageApp.LogMessage("error", fmt.Sprint(errMsg))

		case <-done:
			numGoProg--

			if numGoProg == 0 {
				break DONE
			}
		}
	}

	sysInfo.Info.VersionApp = mc.VersionApp
	sysInfo.Info.IPAddress = mc.ExternalIPAddress
	sysInfo.Info.CurrentDateTime = time.Now().Unix() * 1000
	sysInfo.MessageType = "information"

	formatJSON, err := json.Marshal(sysInfo)
	if err != nil {
		_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
	}

	out <- formatJSON
}
