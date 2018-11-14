package configure

/*
* Описание типа содержащего настройки приложения
* Версия 0.1, дата релиза 12.02.2018
* */

//MothConfig хранит настройки из конфигурационного файла приложения
type MothConfig struct {
	RootDir                   string
	PathMainConfigurationFile string   `json:"pathMainConfigurationFile"`
	AuthenticationToken       string   `json:"authenticationToken"`
	ExternalIPAddress         string   `json:"externalIPAddress"`
	ExternalPort              int      `json:"externalPort"`
	PathCertFile              string   `json:"pathCertFile"`
	PathKeyFile               string   `json:"pathKeyFile"`
	TypeAreaNetwork           int      `json:"typeAreaNetwork"`
	CurrentDisks              []string `json:"currentDisks"`
	PathStorageFilterFiles    string   `json:"pathStorageFilterFiles"`
	RefreshIntervalSysInfo    int      `json:"refreshIntervalSysInfo"`
	PathLogFiles              string   `json:"pathLogFiles"`
}
