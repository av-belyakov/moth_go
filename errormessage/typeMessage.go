package errormessage

import (
	"encoding/json"
	"errors"

	"github.com/gorilla/websocket"
)

/*
*  Коды ошибок:
*
* 400 - неверно сформированный запрос (совсем неверно сформированный запрос)
* 401 - не найдены файлы для экспорта
* 403 - пользователь не авторизован
* 404 - запрашиваемый ресур не найден (когда переданный пользователем messageType не найден)
* 405 - неожиданный метод запроса
* 406 - переданны неверные данные (данные для фильтрации или выгрузке файлов)
* 409 - несовпадение идентификатора процесса
* 410 - превышен максимальный лимит одновременно выполняемых задач
* 412 - условие ложное (когда параметры фильтрации корректны но по ним не найденно ни одного файла)
* 413 - отсутствуют файлы соответствующие заданному временному интервалу
* 414 - ошибка при попытке остановить загрузку файлов
* 415 - превышен лимит количества попыток повторной передачи файлов
*
* 500 - внутренная ошибка сервера (любая внутренняя ошибка сервера)
*
* */

//ErrorMessageSend содержит описание ошибки отправляемой клиенту
type ErrorMessageSend struct {
	MessageType       string `json:"messageType"`
	ErrorCode         int    `json:"errorCode"`
	ErrorMessage      string `json:"errorMessage"`
	TaskIndex         string `json:"taskId"`
	ExternalIPAddress string `json:"serverIp"`
}

//DetailedMessage детальное описание хранимоц структуры ошибок
type DetailedMessage struct {
	Code    int
	Message string
}

//ListErrorMessage содержит список типов ошибок
type ListErrorMessage struct {
	ShortMessage map[string]DetailedMessage
}

//Options содержит опции необходимые для формирования и отправки сообщения об ошибке
type Options struct {
	ErrMsg, RemoteIP, ExternalIP, TaskIndex string
	Wsc                                     *websocket.Conn
}

//SendErrorMessage формирует и отправляет клиенту сообщение об ошибке
func SendErrorMessage(options Options) error {
	var err error
	if options.Wsc == nil {
		err = errors.New("error message sending is not possible, there is no websocket connection handle")
		return err
	}

	listErrorMessage := map[string]DetailedMessage{
		"badRequest":                               DetailedMessage{400, "Bad Request"},
		"filesNotFound":                            DetailedMessage{401, "Files Not Found"},
		"authenticationError":                      DetailedMessage{403, "Forbidden"},
		"clientError":                              DetailedMessage{404, "Page not found"},
		"unexpectedMethod":                         DetailedMessage{405, "Unexpected request method"},
		"unexpectedValue":                          DetailedMessage{406, "Received unexpected value"},
		"noCoincidenceID":                          DetailedMessage{409, "No coincidence process ID"},
		"limitTasks":                               DetailedMessage{410, "Exceeding the maximum limit of tasks"},
		"userDataIncorrect":                        DetailedMessage{412, "Precondition failed"},
		"noFilesMatchingConfiguredInterval":        DetailedMessage{413, "There are no files matching the configured interval"},
		"errorProcessingStopUploadFiles":           DetailedMessage{414, "An error stop the boot files"},
		"exceededNumberCyclesUploadRetransmission": DetailedMessage{415, "You have exceeded the number of cycles of re-transfer files"},
		"serverError":                              DetailedMessage{500, "Internal server error"},
	}

	var errorMessageSend ErrorMessageSend
	errorMessageSend.MessageType = "error"
	errorMessageSend.ExternalIPAddress = options.ExternalIP
	errorMessageSend.ErrorCode = listErrorMessage[options.ErrMsg].Code
	errorMessageSend.ErrorMessage = listErrorMessage[options.ErrMsg].Message
	errorMessageSend.TaskIndex = options.TaskIndex

	errorResponse, err := json.Marshal(errorMessageSend)
	if err != nil {
		return err
	}

	err = options.Wsc.WriteMessage(1, errorResponse)
	if err != nil {
		return err
	}
	return nil
}
