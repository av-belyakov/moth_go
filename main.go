package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-ini/ini"
	"github.com/gorilla/websocket"

	"moth_go/configure"
	"moth_go/processingMessageComingChannel"
	"moth_go/routes"
	"moth_go/saveMessageApp"
	"moth_go/sysInfo"
)

//ListAccessIPAddress хранит разрешенные для соединения ip адреса
type ListAccessIPAddress struct {
	IPAddress []string
}

//SettingsHTTPServer хранит параметры необходимые при взаимодействии с HTTP сервером
type SettingsHTTPServer struct {
	IP, Port, Token string
}

var mc configure.MothConfig
var acc configure.AccessClientsConfigure
var listAccessIPAddress ListAccessIPAddress

//здесь хранится информация о всех задачах по фильтрации
var ift configure.InformationFilteringTask

//здесь хранится информация о всех задачах по выгрузке сет. трафика
var dfi configure.DownloadFilesInformation

//ReadMainConfig читает основной конфигурационный файл и сохраняет данные в MothConfig
func readMainConfig(fileName string, mc *configure.MothConfig) error {
	var err error
	row, err := ioutil.ReadFile(fileName)
	if err != nil {
		return err
	}

	err = json.Unmarshal(row, &mc)
	if err != nil {
		return err
	}

	return err
}

func readSecondaryConfig(mc *configure.MothConfig) error {
	var err error
	fileConf, err := os.Open(mc.PathMainConfigurationFile)
	if err != nil {
		return err
	}
	defer fileConf.Close()

	cfg, err := ini.Load(mc.PathMainConfigurationFile)
	if err != nil {
		return err
	}

	ipAndPort := cfg.Section("moth").Key("moth-local-addr").String()
	ipPort := strings.Split(ipAndPort, ":")

	port, err := strconv.Atoi(ipPort[1])
	if err != nil {
		return err
	}
	areaNetwork, err := strconv.Atoi(cfg.Section("moth").Key("moth-area-network").String())
	if err != nil {
		return err
	}

	var storage string
	arrayStorages := make([]string, 0)
	for i := 0; i < 15; i++ {
		num := strconv.Itoa(i)
		storage = cfg.Section("storage").Key("storage" + num).String()
		if len(storage) == 0 {
			break
		}

		arrayStorages = append(arrayStorages, storage)
	}

	mc.ExternalIPAddress = ipPort[0]
	mc.ExternalPort = port
	mc.AuthenticationToken = cfg.Section("moth").Key("moth-auth-token").String()
	mc.TypeAreaNetwork = areaNetwork
	mc.CurrentDisks = arrayStorages

	return err
}

//HandlerRequest обработчик HTTPS запроса к "/"
func (settingsHTTPServer *SettingsHTTPServer) HandlerRequest(w http.ResponseWriter, req *http.Request) {
	bodyHTTPResponseError := []byte(`<!DOCTYPE html>
		<html lang="en"
		<head><meta charset="utf-8"><title>Server Nginx</title></head>
		<body><h1>Access denied. For additional information, please contact the webmaster.</h1></body>
		</html>`)

	stringToken := ""
	for headerName := range req.Header {
		if headerName == "Token" {
			stringToken = req.Header[headerName][0]
			continue
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Language", "en")

	if req.Method != "GET" {
		http.Error(w, "Method not allowed", 405)
		return
	}

	if (len(stringToken) == 0) || (stringToken != settingsHTTPServer.Token) {
		w.Header().Set("Content-Length", strconv.Itoa(utf8.RuneCount(bodyHTTPResponseError)))

		w.WriteHeader(400)
		w.Write(bodyHTTPResponseError)

		_ = saveMessageApp.LogMessage("error", "missing or incorrect identification token (сlient ipaddress "+req.RemoteAddr+")")
	} else {
		http.Redirect(w, req, "https://"+settingsHTTPServer.IP+":"+settingsHTTPServer.Port+"/wss", 301)

		if !acc.IPAddressIsExist(strings.Split(req.RemoteAddr, ":")[0]) {
			remoteAddr := strings.Split(req.RemoteAddr, ":")[0]

			fmt.Println("GET remote IP ", remoteAddr)

			acc.Addresses[remoteAddr] = &configure.ClientsConfigure{}
		}
	}
}

func serverWss(w http.ResponseWriter, req *http.Request) {
	remoteIP := strings.Split(req.RemoteAddr, ":")[0]
	if !acc.IPAddressIsExist(remoteIP) {
		w.WriteHeader(401)
		_ = saveMessageApp.LogMessage("error", "access for the user with ipaddress "+req.RemoteAddr+" is prohibited")
		return
	}

	/*
		канал для закрытия анонимной go-подпрограммы принимающей данные
		из каналов ChanWebsocketTranssmition и ChanWebsocketTranssmitionBinary
	*/
	chanEndGoroutin := make(chan struct{})

	//канал для останова передачи системной информации
	chanStopSendInfoTranssmition := make(chan struct{})

	//иницилизируем канал для приема информации по скачиваемым файлам
	acc.ChanInfoDownloadTaskGetMoth = make(chan configure.ChanInfoDownloadTask, 5)
	//иницилизируем канал для передачи информации по скачиваемым файлам
	acc.ChanInfoDownloadTaskSendMoth = make(chan configure.ChanInfoDownloadTask, 5)

	var upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
		EnableCompression: false,
		//ReadBufferSize:    1024,
		//WriteBufferSize:   100000000,
		HandshakeTimeout: (time.Duration(1) * time.Second),
	}

	c, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		c.Close()

		_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
	}
	defer func() {
		close(acc.ChanInfoDownloadTaskSendMoth)

		chanEndGoroutin <- struct{}{}

		c.Close()

		//удаляем задачу п овыгрузки файлов
		/*if dfi.HasTaskDownloadFiles(pfrdf.RemoteIP, taskIndex) {
			dfi.DelTaskDownloadFiles(pfrdf.RemoteIP)
		}*/

		//удаляем информацию о соединении из типа acc
		delete(acc.Addresses, remoteIP)
		_ = saveMessageApp.LogMessage("info", "disconnect for IP address "+remoteIP)

		if _, ok := acc.Addresses[remoteIP]; !ok {
			fmt.Println(ok, "--- --- ---- IPADDRESS ", remoteIP, "NOT FOUND, WEBSOCKET DISCONNECT")
		}

		//при разрыве соединения удаляем задачу по скачиванию файлов
		dfi.DelTaskDownloadFiles(remoteIP)

		fmt.Println("websocket disconnect!!!")
		fmt.Println("_!!!_----- COUNT GOROUTINE AFTER:", runtime.NumGoroutine())

	}()

	acc.Addresses[remoteIP].WsConnection = c

	acc.ChanWebsocketTranssmition = make(chan []byte)
	acc.ChanWebsocketTranssmitionBinary = make(chan []byte)

	go func(acc *configure.AccessClientsConfigure) {

	DONE:
		for {
			/*message := <-acc.ChanWebsocketTranssmition
			if _, isExist := acc.Addresses[remoteIP]; isExist {
				if err := acc.Addresses[remoteIP].SendWsMessage(1, message); err != nil {
					_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
				}
			}*/

			select {
			case messageText := <-acc.ChanWebsocketTranssmition:
				if _, isExist := acc.Addresses[remoteIP]; isExist {
					if err := acc.Addresses[remoteIP].SendWsMessage(1, messageText); err != nil {
						_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
					}
				}
			case messageBinary := <-acc.ChanWebsocketTranssmitionBinary:
				if _, isExist := acc.Addresses[remoteIP]; isExist {
					if err := acc.Addresses[remoteIP].SendWsMessage(2, messageBinary); err != nil {
						_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
					}
				}
			case <-chanEndGoroutin:

				chanStopSendInfoTranssmition <- struct{}{}

				break DONE
			}

			/*
				ТАК КАК тип сообщения определяется значениями 1 или 2,
				1 - текст, 2 - бинарный, то необходимо предусмотреть получение
				из нескольких каналов (через SELECT) и организовать выбор типа отправляемого
				сообщения.

				select {
				case messageText := <-acc.ChanWebsocketTranssmition:
					if _, isExist := acc.Addresses[remoteIP]; isExist {
						if err := acc.Addresses[remoteIP].SendWsMessage(1, message); err != nil {
							_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
						}
					}
				case messageBinary := <- acc.ChanWebsocketTranssmitionBinary:
					if _, isExist := acc.Addresses[remoteIP]; isExist {
						if err := acc.Addresses[remoteIP].SendWsMessage(2, message); err != nil {
							_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
						}
					}
				}

				!!! ДОПОЛНИТЕЛЬНО создать два канала
				- для передачи бинарных данных
				- для передачи сообщений об остановке чтения файлов

			*/

			/*select {
			case message := <-acc.ChanWebsocketTranssmition:
				if _, isExist := acc.Addresses[remoteIP]; isExist {
					if err := acc.Addresses[remoteIP].SendWsMessage(1, message); err != nil {
						_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
					}
				}
			default:
			}*/
		}

		fmt.Println("**** STOP GOROUTIN resived chans 'ChanWebsocketTranssmition' and 'ChanWebsocketTranssmitionBinary'")
		fmt.Println("_!!!_ COUNT GOROUTINE:", runtime.NumGoroutine())

	}(&acc)

	if e := recover(); e != nil {
		_ = saveMessageApp.LogMessage("error", fmt.Sprint(e))
	}

	routes.RouteWebSocketRequest(remoteIP, &acc, &ift, &dfi, &mc, chanStopSendInfoTranssmition)
}

//processMsgFilterComingChannel обрабатывает иформацию о фильтрации получаемую из канала
/*func processMsgFilterComingChannel(acc *configure.AccessClientsConfigure, ift *configure.InformationFilteringTask) {
	sendStopMsg := func(taskIndex string, task *configure.TaskInformation, sourceData *configure.ClientsConfigure) {
		MessageTypeFilteringStop := configure.MessageTypeFilteringStop{
			MessageType: "filtering",
			Info: configure.MessageTypeFilteringStopInfo{
				configure.FilterInfoPattern{
					Processing: task.TypeProcessing,
					TaskIndex:  taskIndex,
					IPAddress:  task.RemoteIP,
				},
			},
		}

		fmt.Println("--------------------- FILTERING COMPLETE -------------------")
		fmt.Println(MessageTypeFilteringStop)

		fmt.Println("++++++ job status: ", task.TypeProcessing, ", task ID:", taskIndex, "count files found:", task.CountFilesFound)

		formatJSON, err := json.Marshal(&MessageTypeFilteringStop)
		if err != nil {
			_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
		}

		if _, ok := acc.Addresses[task.RemoteIP]; ok {
			acc.ChanWebsocketTranssmition <- formatJSON
		}

		delete(ift.TaskID, taskIndex)
		_ = saveMessageApp.LogMessage("info", task.TypeProcessing+" of the filter task execution with ID"+taskIndex)
	}

	for msgInfoFilterTask := range acc.ChanInfoFilterTask {
		if task, ok := ift.TaskID[msgInfoFilterTask.TaskIndex]; ok {

			//fmt.Println("====== RESIVED FROM CHAN MSG type processing", msgInfoFilterTask.TypeProcessing, "TYPE PROCESSING SAVE TASK", task.TypeProcessing)
			fmt.Println("====== RESIVED FROM CHAN", task.CountFilesProcessed)

			task.RemoteIP = msgInfoFilterTask.RemoteIP
			task.CountFilesFound = msgInfoFilterTask.CountFilesFound
			task.CountFoundFilesSize = msgInfoFilterTask.CountFoundFilesSize
			task.ProcessingFileName = msgInfoFilterTask.ProcessingFileName
			task.StatusProcessedFile = msgInfoFilterTask.StatusProcessedFile

			task.CountFilesProcessed++
			task.CountCycleComplete++

			if sourceData, ok := acc.Addresses[task.RemoteIP]; ok {

				switch msgInfoFilterTask.TypeProcessing {
				case "execute":
					if (task.TypeProcessing == "stop") || (task.TypeProcessing == "complete") {
						continue
					}
					fmt.Println("++++++ job status: ", task.TypeProcessing, ", task ID:", msgInfoFilterTask.TaskIndex, "count files found:", task.CountFilesFound)

					mtfeou := configure.MessageTypeFilteringExecutedOrUnexecuted{
						MessageType: "filtering",
						Info: configure.MessageTypeFilteringExecuteOrUnexecuteInfo{
							configure.FilterInfoPattern{
								IPAddress:  msgInfoFilterTask.RemoteIP,
								TaskIndex:  msgInfoFilterTask.TaskIndex,
								Processing: msgInfoFilterTask.TypeProcessing,
							},
							configure.FilterCountPattern{
								CountFilesProcessed:   task.CountFilesProcessed,
								CountFilesUnprocessed: task.CountFilesUnprocessed,
								CountCycleComplete:    task.CountCycleComplete,
								CountFilesFound:       msgInfoFilterTask.CountFilesFound,
								CountFoundFilesSize:   msgInfoFilterTask.CountFoundFilesSize,
							},
							configure.InfoProcessingFile{
								FileName:          msgInfoFilterTask.ProcessingFileName,
								DirectoryLocation: msgInfoFilterTask.DirectoryName,
								StatusProcessed:   msgInfoFilterTask.StatusProcessedFile,
							},
						},
					}

					formatJSON, err := json.Marshal(&mtfeou)
					if err != nil {
						_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
					}

					if _, ok := acc.Addresses[task.RemoteIP]; ok {
						acc.ChanWebsocketTranssmition <- formatJSON
					}
				case "complete":
					processingWebsocketRequest.SendMsgFilteringComplite(acc, ift, msgInfoFilterTask.TaskIndex, task)
				case "stop":
					sendStopMsg(msgInfoFilterTask.TaskIndex, task, sourceData)
				}
			}
		}
	}

	fmt.Println("**** STOP GOROUTIN ----'processMsgFilterComingChannel'-----")

}*/

func init() {
	//проверяем наличие tcpdump
	func() {
		stdout, err := exec.Command("sh", "-c", "whereis tcpdump").Output()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		list := strings.Split(string(stdout), " ")

		if !strings.Contains(list[1], "tcpdump") {
			fmt.Println("tcpdump is not found")
			os.Exit(1)
		}
	}()

	var err error
	//читаем основной конфигурационный файл в формате JSON
	err = readMainConfig("config.json", &mc)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	//читаем вспомогательный конфигурационный файл в формате INI
	err = readSecondaryConfig(&mc)
	if err != nil {
		msg := "конфигурационный файл zsensor.conf отсутствует, используем основной конфигурационный файл"
		_ = saveMessageApp.LogMessage("info", msg)
		log.Println(msg)
	}

	acc.Addresses = make(map[string]*configure.ClientsConfigure)
	//иницилизируем канал для передачи системной информации
	acc.ChanInfoTranssmition = make(chan []byte)

	//иницилизируем канал для передачи информации по фильтрации сет. трафика
	acc.ChanInfoFilterTask = make(chan configure.ChanInfoFilterTask, (len(mc.CurrentDisks) * 5))

	//создаем канал генерирующий регулярные запросы на получение системной информации
	ticker := time.NewTicker(time.Duration(mc.RefreshIntervalSysInfo) * time.Second)

	go func() {
		for {
			select {
			case <-ticker.C:

				//fmt.Println("next tick get SYSTEM INFO");

				go sysInfo.GetSystemInformation(acc.ChanInfoTranssmition, &mc)
			}
		}
	}()

	ift.TaskID = make(map[string]*configure.TaskInformation)
	dfi.RemoteIP = make(map[string]*configure.TaskInformationDownloadFiles)

	//обработка информационных сообщений о фильтрации (канал ChanInfoFilterTask)
	go processingMessageComingChannel.ProcessMsgFilterComingChannel(&acc, &ift)

}

func main() {
	var err error
	var settingsHTTPServer SettingsHTTPServer

	settingsHTTPServer.IP = mc.ExternalIPAddress
	settingsHTTPServer.Port = strconv.Itoa(mc.ExternalPort)
	settingsHTTPServer.Token = mc.AuthenticationToken

	/* инициализируем HTTPS сервер */
	log.Println("The HTTPS server is running ipaddress " + settingsHTTPServer.IP + ", port " + settingsHTTPServer.Port + "\n")

	http.HandleFunc("/", settingsHTTPServer.HandlerRequest)
	http.HandleFunc("/wss", serverWss)

	err = http.ListenAndServeTLS(settingsHTTPServer.IP+":"+settingsHTTPServer.Port, mc.PathCertFile, mc.PathKeyFile, nil)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
