package configure

/*
* Описания структур принимаемых JSON сообщений
* Версия 0.1, дата релиза 22.02.2018
* */

//MessageTypeSetting содержит детальную информацию о фильтрации
type MessageTypeSetting struct {
	DateTimeStart     uint64              `json:"dateTimeStart"`
	DateTimeEnd       uint64              `json:"dateTimeEnd"`
	IPAddress         []string            `json:"ipaddress"`
	Network           []string            `json:"network"`
	UseIndexes        bool                `json:"useIndexes"`
	CountIndexesFiles [2]int              `json:"countIndexesFiles"`
	ListFilesFilter   map[string][]string `json:"listFilesFilter"`
}

//MessageTypeFilterInfo содержит общую инфрмацию о фильтрации
type MessageTypeFilterInfo struct {
	Processing string             `json:"processing"`
	TaskIndex  string             `json:"taskIndex"`
	Settings   MessageTypeSetting `json:"settings"`
}

//MessageTypeFilter содержит всю информацию
type MessageTypeFilter struct {
	Info MessageTypeFilterInfo `json:"info"`
}
