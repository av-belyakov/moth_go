package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-ini/ini"
	"github.com/gorilla/websocket"

	"moth_go/configure"
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
var accessClientsConfigure configure.AccessClientsConfigure
var listAccessIPAddress ListAccessIPAddress

var informationFilteringTask configure.InformationFilteringTask

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

		if !accessClientsConfigure.IPAddressIsExist(strings.Split(req.RemoteAddr, ":")[0]) {
			remoteAddr := strings.Split(req.RemoteAddr, ":")[0]

			fmt.Println("GET reomte IP ", remoteAddr)

			accessClientsConfigure.Addresses[remoteAddr] = &configure.ClientsConfigure{}
		}
	}
}

func serverWss(w http.ResponseWriter, req *http.Request) {
	remoteIP := strings.Split(req.RemoteAddr, ":")[0]
	if !accessClientsConfigure.IPAddressIsExist(remoteIP) {
		w.WriteHeader(401)
		_ = saveMessageApp.LogMessage("error", "access for the user with ipaddress "+req.RemoteAddr+" is prohibited")
		return
	}

	var upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
		EnableCompression: false,
		//ReadBufferSize:    1024,
		//WriteBufferSize:   100000000,
		HandshakeTimeout: (time.Duration(5) * time.Second),
	}

	c, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		c.Close()
		_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
	}
	defer c.Close()

	accessClientsConfigure.Addresses[remoteIP].WsConnection = c

	routes.RouteWebSocketRequest(remoteIP, &accessClientsConfigure, &informationFilteringTask, &mc)
}

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
		msg := "файл zsensor.conf отсутствует, используем основной конфигурационный файл"
		_ = saveMessageApp.LogMessage("info", msg)
		log.Println(msg)
	}

	accessClientsConfigure.Addresses = make(map[string]*configure.ClientsConfigure)
	//иницилизируем канал для передачи системной информации
	accessClientsConfigure.ChanInfoTranssmition = make(chan []byte)
	//иницилизируем канал для передачи информации по фильтрации сет. трафика
	accessClientsConfigure.ChanInfoFilterTask = make(chan configure.ChanInfoFilterTask, (len(mc.CurrentDisks) * 5))

	//создаем канал генерирующий регулярные запросы на получение системной информации
	ticker := time.NewTicker(time.Duration(mc.RefreshIntervalSysInfo) * time.Second)

	go func() {
		for {
			select {
			case <-ticker.C:
				go sysInfo.GetSystemInformation(accessClientsConfigure.ChanInfoTranssmition, &mc)
			}
		}
	}()

	informationFilteringTask.TaskID = make(map[string]*configure.TaskInformation)
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
