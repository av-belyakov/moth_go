package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-ini/ini"
	"github.com/gorilla/websocket"

	"moth_go/configure"
	"moth_go/processingmessagecomingchannel"
	"moth_go/routes"
	"moth_go/savemessageapp"
	"moth_go/sysinfo"
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

func getVersionApp(mc *configure.MothConfig) (string, error) {
	content, err := ioutil.ReadFile(mc.RootDir + "README.md")
	if err != nil {
		return "", err
	}

	//Application Moth_go, v1.35
	//rx := regexp.MustCompile(`Moth_go\s(version=\d+\\.\d+)`)
	pattern := `^Application\sMoth_go,\sv\d+\.\d+`
	rx := regexp.MustCompile(pattern)
	numVersion := rx.FindString(string(content))

	if len(numVersion) == 0 {
		return "version not found", nil
	}

	s := strings.Split(numVersion, " ")

	return s[2], nil
}

//HandlerRequest обработчик HTTPS запроса к "/"
func (settingsHTTPServer *SettingsHTTPServer) HandlerRequest(w http.ResponseWriter, req *http.Request) {
	//инициализируем функцию конструктор для записи лог-файлов
	saveMessageApp := savemessageapp.New()

	bodyHTTPResponseError := []byte(`<!DOCTYPE html>
		<html lang="en"
		<head><meta charset="utf-8"><title>Server Nginx</title></head>
		<body><h1>Access denied. For additional information, please contact the webmaster.</h1></body>
		</html>`)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Language", "en")

	if req.Method != "GET" {
		http.Error(w, "Method not allowed", 405)
		return
	}

	//fmt.Println("---- http header accepted from the user ----")

	stringToken := ""
	for headerName := range req.Header {

		//fmt.Printf("Header: %v, value: %v\n", headerName, req.Header[headerName][0])

		if headerName == "Token" {
			stringToken = req.Header[headerName][0]
			continue
		}
	}

	//fmt.Println("\ttokent is valide: ", (stringToken == settingsHTTPServer.Token))

	if (len(stringToken) == 0) || (stringToken != settingsHTTPServer.Token) {
		w.Header().Set("Content-Length", strconv.Itoa(utf8.RuneCount(bodyHTTPResponseError)))

		w.WriteHeader(400)
		w.Write(bodyHTTPResponseError)

		_ = saveMessageApp.LogMessage("error", "missing or incorrect identification token (сlient ipaddress "+req.RemoteAddr+")")
	} else {
		http.Redirect(w, req, "https://"+settingsHTTPServer.IP+":"+settingsHTTPServer.Port+"/wss", 301)

		if !acc.IPAddressIsExist(strings.Split(req.RemoteAddr, ":")[0]) {
			remoteAddr := strings.Split(req.RemoteAddr, ":")[0]

			acc.Addresses[remoteAddr] = &configure.ClientsConfigure{}

			//log.Printf("client configure %v\n", acc)
		}
	}
}

func serverWss(w http.ResponseWriter, req *http.Request) {
	//инициализируем функцию конструктор для записи лог-файлов
	saveMessageApp := savemessageapp.New()

	remoteIP := strings.Split(req.RemoteAddr, ":")[0]

	fmt.Println("remote ip client is:", remoteIP)

	if !acc.IPAddressIsExist(remoteIP) {
		//log.Println("access for the user with ipaddress " + req.RemoteAddr + " is prohibited")

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
		//log.Printf("connection close, error %v\n", err)

		c.Close()

		_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))

		return
	}
	defer func() {
		//log.Printf("DEFER close function")

		close(acc.ChanInfoDownloadTaskGetMoth)
		close(acc.ChanInfoDownloadTaskSendMoth)

		chanEndGoroutin <- struct{}{}
	}()

	//log.Printf("the connection is established")

	acc.Addresses[remoteIP].WsConnection = c

	acc.ChanWebsocketTranssmition = make(chan []byte)
	acc.ChanWebsocketTranssmitionBinary = make(chan []byte)

	go func(acc *configure.AccessClientsConfigure) {
	DONE:
		for {
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
		}

		//		fmt.Println("_!!!_ COUNT GOROUTINE:", runtime.NumGoroutine())
	}(&acc)

	if e := recover(); e != nil {
		_ = saveMessageApp.LogMessage("error", fmt.Sprint(e))
	}

	routes.RouteWebSocketRequest(remoteIP, &acc, &ift, &dfi, &mc, chanStopSendInfoTranssmition)
}

func init() {
	//инициализируем функцию конструктор для записи лог-файлов
	saveMessageApp := savemessageapp.New()

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

	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}

	//читаем основной конфигурационный файл в формате JSON
	err = readMainConfig(dir+"/config.json", &mc)
	if err != nil {
		fmt.Println("Error reading configuration file, ", err)
		os.Exit(1)
	}

	mc.RootDir = dir + "/"

	mc.PathKeyFile = mc.RootDir + mc.PathKeyFile
	mc.PathCertFile = mc.RootDir + mc.PathCertFile

	//читаем вспомогательный конфигурационный файл в формате INI
	err = readSecondaryConfig(&mc)
	if err != nil {
		msg := "конфигурационный файл zsensor.conf отсутствует, используем основной конфигурационный файл"
		_ = saveMessageApp.LogMessage("info", msg)
		log.Println(msg)
	}

	//получаем номер версии приложения
	version, err := getVersionApp(&mc)
	if err != nil {
		_ = saveMessageApp.LogMessage("err", "it is impossible to obtain the version number of the application")
	}
	mc.VersionApp = version

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
				go sysinfo.GetSystemInformation(acc.ChanInfoTranssmition, &mc)
			}
		}
	}()

	ift.TaskID = make(map[string]*configure.TaskInformation)
	dfi.RemoteIP = make(map[string]*configure.TaskInformationDownloadFiles)

	//обработка информационных сообщений о фильтрации (канал ChanInfoFilterTask)
	go processingmessagecomingchannel.ProcessMsgFilterComingChannel(&acc, &ift)
}

func main() {
	var err error
	var settingsHTTPServer SettingsHTTPServer

	settingsHTTPServer.IP = mc.ExternalIPAddress
	settingsHTTPServer.Port = strconv.Itoa(mc.ExternalPort)
	settingsHTTPServer.Token = mc.AuthenticationToken

	/* инициализируем HTTPS сервер */
	log.Println("The HTTPS server is running on ip address " + settingsHTTPServer.IP + ", port " + settingsHTTPServer.Port + "\n")

	http.HandleFunc("/", settingsHTTPServer.HandlerRequest)
	http.HandleFunc("/wss", serverWss)

	err = http.ListenAndServeTLS(settingsHTTPServer.IP+":"+settingsHTTPServer.Port, mc.PathCertFile, mc.PathKeyFile, nil)

	log.Println(err)
	os.Exit(1)
}
