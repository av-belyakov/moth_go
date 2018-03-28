package sysInfo

import (
	"encoding/json"
	"fmt"
	"time"

	"moth_go/configure"
	"moth_go/saveMessageApp"
)

//GetSystemInformation позволяет получить системную информацию
func GetSystemInformation(out chan<- []byte, mc *configure.MothConfig) {
	var sysInfo configure.SysInfo

	fmt.Println("\nINFO: function GetSystemInformation STARTING...")

	var done = make(chan struct{})
	var errorMessage = make(chan error)
	defer func() {
		close(done)
		close(errorMessage)
	}()

	//временной интервал файлов хранящихся на дисках
	go sysInfo.CreateFilesRange(done, errorMessage, mc.CurrentDisks)
	select {
	case <-done:
		break
	case <-errorMessage:
		_ = saveMessageApp.LogMessage("error", fmt.Sprint(errorMessage))
		break
	}
	fmt.Println("CreateFilesRange COMPLETE")

	//загрузка оперативной памяти
	if err := sysInfo.CreateRandomAccessMemory(); err != nil {
		_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
	}
	fmt.Println("RandomAccessMemory COMPLETE")

	//загрузка ЦПУ
	if err := sysInfo.CreateLoadCPU(); err != nil {
		_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
	}
	fmt.Println("CreateLoadCPU COMPLETE")

	//нагрузка на сетевых интерфейсах
	go sysInfo.CreateLoadNetworkInterface(done, errorMessage)
	select {
	case <-done:
		break
	case <-errorMessage:
		fmt.Println(errorMessage)
		break
	}
	fmt.Println("CreateLoadNetworkInterface COMPLETE")

	//свободное дисковое пространство
	if err := sysInfo.CreateDiskSpace(); err != nil {
		_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
	}
	fmt.Println("CreateDiskSpace COMPLETE")

	sysInfo.Info.IPAddress = mc.ExternalIPAddress
	sysInfo.Info.CurrentDateTime = time.Now().Unix() * 1000
	sysInfo.MessageType = "information"

	formatJSON, err := json.Marshal(sysInfo)
	if err != nil {
		_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
	}

	out <- formatJSON
}
