package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	config "mdata/configs"
	dom "mdata/internal/domain"
	"mdata/internal/repository"
	"mdata/internal/utils"
	log "mdata/pkg/logging"
)

//const api1CZupPing = "http://md:masteR+Data2021@srv-web1:8086/EmployeeData/hs/VK_EmployeeData/ping"
//const api1CZupAllEmployees = "http://md:masteR+Data2021@srv-web1:8086/EmployeeData/hs/VK_EmployeeData/get_all_users"
//const putTo1CZupEmailAdress = "http://md:masteR+Data2021@srv-web1:8086/mdzup-load/hs/mdzup-load-email/put-email"
//const putTo1CCreateUserAdress = "http://srv-web1:8086/mdzup-load/hs/mdzup-load-email/put-email" //  TODO
//const api1CZupAllDepartaments = "http://md:masteR+Data2021@srv-web1:8086/EmployeeData/hs/VK_EmployeeData/get_departaments"

const client1CTimeout = 20

//*******************************************
// ping - метод. Доступность сервиса MD
func PingMasterData(w http.ResponseWriter, r *http.Request) {
	resp := PingMD()
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(resp))
}

//*******************************************
// ping - метод. Получим доступность базы 1С:ЗУП
func PingZup(w http.ResponseWriter, r *http.Request) {
	// ctx, cancel := context.WithTimeout(r.Context(), 10*time.Millisecond)
	// defer cancel()
	// req, err := http.NewRequestWithContext(ctx, http.MethodGet, api1CZupPing, nil)
	req, err := http.NewRequest(http.MethodGet, config.GetApi1CZupPing(), nil)
	if err != nil {
		log.Error("rest: handlers.PingZup: no request: %v", err)
	}

	client := http.Client{Timeout: client1CTimeout * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		strError := fmt.Sprintf("rest: handlers.PingZup: no http client: %v", err)
		log.Error(strError)
		w.WriteHeader(http.StatusNotFound) //StatusRequestTimeout
		w.Write([]byte("1C:ZUP is not available: " + strError))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		log.Error("rest: handlers.PingZup: 1C:ZUP is not available: 404 Not found")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("1C:ZUP is not available"))
		return
	}

	strZupPing, err := ZupPing(resp.Body)
	if err != nil {
		log.Error("rest error: ", err)
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("handlers.PingZup: I'v got strZupPing:\n %v", strZupPing)))
}

func PingDB(ins *repository.PostgreInstance) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		response, err := repository.DBPing(ins)
		if err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(err.Error()))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprint(response, "OK, I'm ready to read and write from mdataDB")))
	}
}

//*******************************************
// ping - метод. Получим всех пользователей 1С:ЗУП
func PingFromZmupAllUsers(w http.ResponseWriter, r *http.Request) {
	// create request
	req, err := http.NewRequest(http.MethodGet, config.GetApi1CZupAllEmployees(), nil)
	if err != nil {
		log.Error("rest: handlers.PingEromZupAllEmployees: no request: ", err)
	}
	// create client, do request and get response
	client := http.Client{Timeout: client1CTimeout * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		strError := fmt.Sprintf("rest: handlers.PingEromZupAllEmployees: no http client: %v", err)
		log.Error(strError)
		w.WriteHeader(http.StatusNotFound) //StatusRequestTimeout
		w.Write([]byte("1C:ZUP is not available: " + strError))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		log.Error("rest: handlers.PingEromZupAllEmployees: 1C:ZUP is not available: 404 Not found")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("1C:ZUP is not available"))
		return
	}

	// have got body of response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error("rest: handlers.PingEromZupAllEmployees: io.ReadAll(Body)", err)
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("rest: handlers.PingFromZmupAllUsers: I'v got strZupPing:\n %v", string(body))))
}

//*******************************************
// Запишем в БД всех user-ов из ЗУП-а, по пути получив их email из AD
func RestHandleZupWriteAllUsers(ins *repository.PostgreInstance) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// request to get user data from 1C:ZUP
		req, err := http.NewRequest(http.MethodGet, config.GetApi1CZupAllEmployees(), nil)
		if err != nil {
			log.Error("handlers.RestHandleZupWriteAllUsers: no request: %v", err)
		}
		// create client, do request and get response with user data from 1C:ZUP
		client := http.Client{Timeout: client1CTimeout * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			strError := fmt.Sprintf("rest: handlers.RestHandleZupWriteAllUsers: no http client: %v", err)
			log.Error(strError)
			w.WriteHeader(http.StatusNotFound) //StatusRequestTimeout
			w.Write([]byte("1C:ZUP is not available: " + strError))
			return
		}
		defer resp.Body.Close()

		// have got body of response
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Error("handlers.RestHandleZupWriteAllUsers: io.ReadAll(Body) %v", err)
		}

		// handling user data and write to db
		strAnswer, err := handleZupWriteAllUsers(ins, body)
		if err != nil {
			log.Error("handlers.RestHandleZupWriteAllUsers handle bodi error: %v", err)
		}
		log.Info(strAnswer)
		// must return "w" - response ("ok", "200" or not)
	}
}

// Testing-Debug. Запишем в БД одного user-а из запроса (из Postman-а), по пути получив его email из AD
func RestHandleDebugWriteOneUser(ins *repository.PostgreInstance) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			log.Error("handlers.RestHandleDebugWriteOneUser: io.ReadAll(Body)", err)
			http.Error(w, "500 - Something bad happened!", 500)
			return
		}
		defer r.Body.Close()

		strAnswer, err := handleZupWriteAllUsers(ins, body)
		if err != nil {
			log.Error("handlers.RestHandleDebugWriteOneUser handle bodi error: %v", err)
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(err.Error()))
			return
		}
		log.Info(strAnswer)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("200 - ok!"))
		// must return "w" - response ("ok", "200" or not)
	}
}

//*******************************************
// Отправляем в 1С:ЗУП email одного захардкоженного пользователя.
func RestPutOneUserEmailToZup(w http.ResponseWriter, r *http.Request) {

	user := dom.User{
		UserGUID:  "6040e284-e021-11e0-82ff-00237de17aae", //"339619FF-3386-3311-332D-33CF4FC964FF",
		UserName:  "Кабанов Алексей Иванович",
		UserID:    "08337",
		UserEmail: "akabanov@vodokanal-nn.ru",
	}

	var sliceUsers = new(dom.AGUsers)
	sliceUsers.Users = append(sliceUsers.Users, user)

	// initialize http client
	client := http.Client{Timeout: client1CTimeout * time.Second}

	// marshal User to json
	json, err := json.Marshal(sliceUsers)
	fmt.Printf("sliceUsers: %v\n", sliceUsers)
	fmt.Printf("json: %v\n", json)
	fmt.Printf("string(json): %v\n", string(json))
	fmt.Printf("json: %v\n", []byte(string(json)))
	if err != nil {
		log.Error("handlers.PutOneUserEmailToZup handl json.Marshal(user) error:", err)
	}

	// set the HTTP method, url, and request body
	req, err := http.NewRequest(http.MethodPut, config.GetPutTo1CZupEmailAdress(), bytes.NewBuffer(json))
	if err != nil {
		log.Error("handlers.PutOneUserEmailToZup handl http.NewRequest error:", err)
	}

	// set the request header Content-Type for json
	req.Header.Set("Content-Type", "application/json") //; charset=utf-8")
	resp, err := client.Do(req)
	if err != nil {
		strError := fmt.Sprintf("rest: handlers.PutOneUserEmailToZup handl client.Do(req) error: %v", err)
		log.Error(strError)
		w.WriteHeader(http.StatusNotFound) //StatusRequestTimeout
		w.Write([]byte("1C:ZUP is not available: " + strError))
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error("handlers.PutOneUserEmailToZup handl io.ReadAll(resp.Body) error:", err)
	}

	if resp.StatusCode != 200 {
		respError := fmt.Errorf("rest: handlers.RestPutOneUserEmailToZup: %v", string(body))
		log.Error(respError.Error())
		http.Error(w, respError.Error(), resp.StatusCode)
		return
	}
	strResp := fmt.Errorf("PutUsersEmailsToZup - %v", resp.Status)
	fmt.Fprint(w, strResp)
}

//*******************************************
// Отправляем в 1С:ЗУП email всех пользователей для обновления email
func RestPutAllUsersWithEmailToZup(ins *repository.PostgreInstance) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body := make([]byte, 0)
		// (после успешной - ??? загрузки в 1С:ЗУП), отправим log-файл в бухгалтерию:
		defer utils.SendEmailToBuch(ins)

		// получаем (формируем) весь массив, который будем передавать
		reasonID := 1                                                                          // для идентификации того, что нам нужны user-ы для выгрузки именно в 1С:ЗУП
		jsonSliceOfByte, ptrExchMap, _, err := getUsersArrayWithEmailToPUTQuery(ins, reasonID) // в мапе (ptrExchMap) получаем id отправляемой строки и attempt_count для понимания, куда (в какую строку) записывать ошибку или 'ok'
		if err != nil {
			errorString := "rest handlers.RestPutAllUsersWithEmailToZup: при выборке данных к обмену произошла ошибка: " + err.Error()
			log.Info(errorString)
			http.Error(w, errorString, 500)
			return // нет данных к обмену
		}
		if len(jsonSliceOfByte) == 0 {
			errorString := "rest handlers.RestPutAllUsersWithEmailToZup: нет данных к выгрузке в 1С:ЗУП"
			log.Info(errorString)
			http.Error(w, errorString, http.StatusNoContent)                             // 204
			go putRequestFrom1CLoadingPrepare(ins, body, ptrExchMap, errorString+" 204") // TODO
			return                                                                       // нет данных к обмену
		}

		// готовим запрос в 1С:ЗУП
		req, err := http.NewRequest("PUT", config.GetPutTo1CZupEmailAdress(), bytes.NewBuffer(jsonSliceOfByte))
		if err != nil {
			errorString := "rest handlers.RestPutAllUsersWithEmailToZup: http.NewRequest error: " + err.Error()
			log.Error(errorString)
			http.Error(w, errorString, 400)
			go putRequestFrom1CLoadingPrepare(ins, body, ptrExchMap, errorString+" 400")
			return
		}
		req.Header.Set("X-Custom-Header", "mdreqwest")
		req.Header.Set("Content-Type", "application/json")

		// отправляем запрос
		client := http.Client{Timeout: client1CTimeout * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			errorString := "rest handlers.RestPutAllUsersWithEmailToZup client.Do(req) error: " + err.Error()
			log.Error(errorString)
			http.Error(w, errorString, http.StatusForbidden) // 403
			go putRequestFrom1CLoadingPrepare(ins, body, ptrExchMap, errorString+" 403")
			return
		}
		defer resp.Body.Close()

		// читаем ответ
		body, err = io.ReadAll(resp.Body)
		if err != nil {
			errorString := "handlers.PutOneUserEmailToZup handl io.ReadAll(resp.Body) error: " + err.Error()
			log.Error(errorString)
			http.Error(w, errorString, http.StatusForbidden) // 403
			go putRequestFrom1CLoadingPrepare(ins, body, ptrExchMap, errorString+" 403")
			return
		}

		// при возникновении ошибок
		if resp.StatusCode == 404 {
			errorString := "rest: handlers.RestPutAllUsersWithEmailToZup: 1C:ZUP is not available: 404 Not found"
			log.Error(errorString)
			//w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("1C:ZUP is not available"))
			http.NotFound(w, r)
			go putRequestFrom1CLoadingPrepare(ins, body, ptrExchMap, errorString)
			return
		} else if resp.StatusCode == 500 {
			errorString := "rest: handlers.RestPutAllUsersWithEmailToZup: 500 Internal server error"
			log.Error(errorString)
			//w.WriteHeader(http.StatusInternalServerError)
			//w.Write([]byte("500 Internal server error"))
			http.Error(w, "500 Internal server error", 500)
			go putRequestFrom1CLoadingPrepare(ins, body, ptrExchMap, errorString)
			return
		} else if resp.StatusCode != 200 {
			respError := fmt.Errorf("rest: handlers.RestPutAllUsersWithEmailToZup: %v error descr: %v", resp.Status, string(body))
			log.Error(respError.Error())
			http.Error(w, resp.Status, resp.StatusCode)
			go putRequestFrom1CLoadingPrepare(ins, body, ptrExchMap, respError.Error())
			return
		}

		log.Info("rest: handlers.RestPutAllUsersWithEmailToZup: %d users and their emails have sent to 1C:ZUP; resp.StatusCode: %d", len(jsonSliceOfByte), resp.StatusCode)
		// если до сюда дошли, то обмен состоялся. Запишем это в табл. "exchanges"
		go putRequestFrom1CLoadingPrepare(ins, body, ptrExchMap, config.Response200ok)
	}
}

//------------------------------------------------------------
// Отправляем в 1С:CreateUser даные новых сотрудников (users) для создания для них пользователей в 1С
func RestPutNewUsersWithEmailToCreate1CAccounts(ins *repository.PostgreInstance) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var errorString string
		var usersToExchangeSlice []dom.User // для формирования письма
		body := make([]byte, 0)

		// (после успешной/не успешной загрузки в 1С:CreateUser), отправим log админам 1С,
		// 	если есть ошибка или ответ от 1С <> 200, а если ответ от 1С = 200, то готовить письмо админам будем в другом месте.
		errorCode := 0
		defer func() {
			if len(usersToExchangeSlice) > 0 {
				if errorCode != 200 {
					go utils.SendEmailTo1CAdminsOtherErrors(ins, usersToExchangeSlice, errorString)
				}
			}
		}()

		// получаем (формируем) весь массив, который будем передавать
		reasonID := 2                                                                                             // для идентификации того, что нам нужны user-ы для выгрузки именно в 1С:CreateUser для создания новых
		jsonSliceOfByte, ptrExchMap, usersToExchangeSlice, err := getUsersArrayWithEmailToPUTQuery(ins, reasonID) // в мапе (ptrExchMap) получаем id отправляемой строки и attempt_count для понимания, куда (в какую строку) записывать ошибку или 'ok'
		if err != nil {
			errorString = "rest handlers.RestPutNewUsersWithEmailToCreate1CAccounts: при выборке данных к обмену произошла ошибка: " + err.Error()
			log.Info(errorString)
			http.Error(w, errorString, 500)
			errorCode = 500
			return // нет данных к обмену
		}
		if len(jsonSliceOfByte) == 0 {
			errorString = "rest handlers.RestPutNewUsersWithEmailToCreate1CAccounts: нет данных к выгрузке в 1С:CreateUser"
			log.Info(errorString)
			http.Error(w, errorString, http.StatusNoContent)
			go putRequestFrom1CLoadingPrepare(ins, body, ptrExchMap, errorString+" 204")
			errorCode = 204
			return
		}

		// готовим запрос в 1С:CreateUser
		reqAdress := config.GetPutTo1CCreateUserAdress()
		req, err := http.NewRequest("PUT", reqAdress, bytes.NewBuffer(jsonSliceOfByte))
		if err != nil {
			errorString = "rest handlers.RestPutNewUsersWithEmailToCreate1CAccounts: http.NewRequest error: " + err.Error()
			log.Error(errorString)
			http.Error(w, errorString, 400)
			go putRequestFrom1CLoadingPrepare(ins, body, ptrExchMap, errorString+" 400")
			errorCode = 400
			return
		}
		req.SetBasicAuth(config.GetPutTo1CCreateUserLogin(), config.GetPutTo1CCreateUserPass()) // TODO перенести в config
		req.Header.Set("X-Custom-Header", "mdrequest")
		req.Header.Set("Content-Type", "application/json")

		// отправляем запрос
		client := http.Client{Timeout: client1CTimeout * 10000 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			errorString = "rest handlers.RestPutNewUsersWithEmailToCreate1CAccounts client.Do(req) error: " + err.Error()
			log.Error(errorString)
			http.Error(w, errorString, http.StatusForbidden) // 403
			go putRequestFrom1CLoadingPrepare(ins, body, ptrExchMap, errorString+" 403")
			errorCode = 403
			return
		}
		defer resp.Body.Close()

		// читаем ответ
		body, err = io.ReadAll(resp.Body)
		if err != nil {
			errorString = "rest handlers.RestPutNewUsersWithEmailToCreate1CAccounts handle io.ReadAll(resp.Body) error: " + err.Error()
			log.Error(errorString)
			http.Error(w, errorString, http.StatusForbidden) // 403
			go putRequestFrom1CLoadingPrepare(ins, body, ptrExchMap, errorString+" 403")
			errorCode = 403
			return
		}

		// при возникновении ошибок
		if resp.StatusCode == 404 {
			errorString = "rest: handlers.RestPutNewUsersWithEmailToCreate1CAccounts: 1C:CreateUser is not available: 404 Not found"
			log.Error(errorString)
			//w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("1C:CreateUser is not available"))
			http.NotFound(w, r)
			go putRequestFrom1CLoadingPrepare(ins, body, ptrExchMap, errorString)
			errorCode = 404
			return
		} else if resp.StatusCode == 500 {
			errorString = "rest: handlers.RestPutNewUsersWithEmailToCreate1CAccounts: 500 Internal server error"
			log.Error(errorString)
			http.Error(w, "500 Internal server error", 500)
			go putRequestFrom1CLoadingPrepare(ins, body, ptrExchMap, errorString)
			errorCode = 500
			return
		} else if resp.StatusCode != 200 {
			respError := fmt.Errorf("rest: handlers.RestPutNewUsersWithEmailToCreate1CAccounts: %v error descr: %v", resp.Status, string(body))
			errorString = respError.Error()
			log.Error(errorString)
			http.Error(w, resp.Status, resp.StatusCode)
			go putRequestFrom1CLoadingPrepare(ins, body, ptrExchMap, errorString)
			errorCode = 1
			return
		}

		log.Info("rest: handlers.RestPutNewUsersWithEmailToCreate1CAccounts: %d users have sent to 1C:CreateUser for accountings; resp.StatusCode: %d", len(jsonSliceOfByte), resp.StatusCode)
		// если до сюда дошли, то обмен состоялся. Запишем это в табл. "exchanges"
		errorCode = 200
		go putRequestFrom1CLoadingPrepare(ins, body, ptrExchMap, config.Response200ok) //// TODO вернуть go !!!!!!!!!!!!!!!!!
	}
}

//------------------------------------------------------------
// все аттрибуты
// Отдаем данные всех работающих (актуальных) сотрудников (все аттрибуты)
func RestSendActEmployeesAllAttributes(ins *repository.PostgreInstance) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// получаем весь массив сотрудников, который будем возвращать
		jsonSliceOfEmployees, err := GetActEmployeesAllAttributes(ins)
		if err != nil {
			http.Error(w, "500 - Something bad happened!", 500)
			w.Write([]byte(err.Error()))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(jsonSliceOfEmployees)
	}
}

// Отдаем данные всех уволенных сотрудников (все аттрибуты)
func RestSendUsersFiredFromAllAttributes(ins *repository.PostgreInstance) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var dateFrom time.Time = time.Now()

		params := r.URL.Query()
		if len(params) < 1 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("There are no parametrs. Please enter right parametr"))
			return
		}
		if len(params) > 1 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Too many parametrs. Please enter right parametr"))
			return
		}

		paramFrom := strings.TrimSpace(params.Get("from"))
		if paramFrom != "" {
			layout := "2006-01-02"

			matched, err := regexp.MatchString(`\d\d\d\d-\d\d-\d\d`, paramFrom)
			if err != nil {
				w.WriteHeader(http.StatusForbidden)
				w.Write([]byte("Somthing wrong. Please try again"))
				return
			}
			if !matched {
				w.WriteHeader(http.StatusForbidden)
				w.Write([]byte("Wrong format. Please try again"))
				return
			}

			dateFrom, err = time.Parse(layout, paramFrom)
			if err != nil {
				w.WriteHeader(http.StatusForbidden)
				w.Write([]byte("Somthing wrong. Please try again"))
				return
			}
		} else {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Wrong parametr. Please enter right parametr"))
			return
		}

		// получаем весь массив сотрудников, который будем возвращать
		jsonSliceOfEmployees, err := GetFiredEmployeesUsersAllAttributes(ins, dateFrom)
		if err != nil {
			http.Error(w, "500 - Something bad happened!", 500)
			w.Write([]byte(err.Error()))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(jsonSliceOfEmployees)
	}
}

// Отдаем данные всех работающих (актуальных) сотрудников, у которых есть email-ы (все аттрибуты)
func RestSendAllEmailEmployeesAllAttributes(ins *repository.PostgreInstance) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// получаем весь массив сотрудников, который будем возвращать
		jsonSliceOfEmployees, err := GetAllEmailEmployeesAllAttributes(ins)
		if err != nil {
			http.Error(w, "500 - Something bad happened!", 500)
			w.Write([]byte(err.Error()))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(jsonSliceOfEmployees)
	}
}

// Отдаем данные одного/группы работающих (актуальных) пользователя(-лей) по одному параметру (обрабатывается как like) (все аттрибуты)
func RestSendEmployeeAllAttributes(ins *repository.PostgreInstance) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// либо по табельному номеру, либо по ФИО:
		t := r.URL.Query()
		if len(t) < 1 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("There are no parametrs."))
			return
		}
		if len(t) > 1 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Too many parametrs."))
			return
		}

		tabno := strings.TrimSpace(r.URL.Query().Get("tabno")) // r.FormValue("tabno")
		userName := strings.TrimSpace(r.URL.Query().Get("name"))
		if tabno != "" {
			// получаем весь массив сотрудников, который будем возвращать, т.к. параметр обрабатывается как like
			jsonSliceOfUsersByTabNo := GetUsersByTabNoAllAttributes(ins, tabno) /////-----------------------------
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write(jsonSliceOfUsersByTabNo)
			return
		} else if userName != "" {
			// получаем весь массив сотрудников, который будем возвращать, т.к. параметр обрабатывается как like
			jsonSliceOfUsersByName := GetUsersByNameAllAttributes(ins, userName)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write(jsonSliceOfUsersByName)
			return
		} else {
			http.NotFound(w, r)
			return
		}
	}
}

//------------------------------------------------------------
// облегчённые аттрибуты
// Отдаем данные всех работающих (актуальных) сотрудников (облегчённые аттрибуты)
func RestSendAllEmployeesLightVersionAttributes(ins *repository.PostgreInstance) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// получаем весь массив сотрудников, который будем возвращать
		jsonSliceOfEmployees, err := GetAllEmployeesLightVersionAttributes(ins)
		if err != nil {
			http.Error(w, "500 - Something bad happened!", 500)
			w.Write([]byte(err.Error()))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(jsonSliceOfEmployees)
	}
}

// Отдаем данные всех работающих (актуальных) сотрудников, у которых есть email-ы (облегчённые аттрибуты)
func RestSendAllEmailEmployeesLightVersionAttributes(ins *repository.PostgreInstance) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// получаем весь массив сотрудников, который будем возвращать
		jsonSliceOfEmployees, err := GetAllEmailEmployeesLightVersionAttributes(ins)
		if err != nil {
			http.Error(w, "500 - Something bad happened!", 500)
			w.Write([]byte(err.Error()))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(jsonSliceOfEmployees)
	}
}

type delayStruct struct {
	delayMap map[string]time.Time
	mu       sync.RWMutex
}

func delayStructInit() *delayStruct {
	return &delayStruct{
		delayMap: make(map[string]time.Time, 0),
	}
}

func (ds *delayStruct) writeTime(comp string, t time.Time) {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	ds.delayMap[comp] = t
}

func (ds *delayStruct) getTime(comp string) (time.Time, bool) {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	t, ok := ds.delayMap[comp]
	if !ok {
		t = time.Time{}
	}
	return t, ok
}

func (ds *delayStruct) delRecord(comp string) {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	delete(ds.delayMap, comp)
}

var globalDelayMap = delayStructInit()

// Отдаем данные одного/группы работающих (актуальных) пользователя(-лей) по одному параметру (обрабатывается как like) (облегчённые аттрибуты)
func RestSendEmployeeLightVersionAttributes(ins *repository.PostgreInstance) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// либо по табельному номеру, либо по ФИО:
		t := r.URL.Query()
		if len(t) < 1 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("There are no parametrs."))
			return
		}
		if len(t) > 1 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Too many parametrs."))
			return
		}

		tabno := strings.TrimSpace(r.URL.Query().Get("tabno"))
		userName := strings.TrimSpace(r.URL.Query().Get("name"))
		if tabno != "" {
			// получаем весь массив сотрудников, который будем возвращать, т.к. параметр обрабатывается как like
			jsonSliceOfUsersByTabNo := GetUsersByTabNoLightVersionAttributes(ins, tabno) /////-----------------------------
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write(jsonSliceOfUsersByTabNo)
			return
		} else if userName != "" {

			// request from 1C
			if strings.Contains(r.Header["User-Agent"][0], "Enterprise") {
				// TODO LRU Cash
				remAddr := r.RemoteAddr
				assertLenRemAddr := len(remAddr) - 6
				remAddr = remAddr[:assertLenRemAddr]
				t, ok := globalDelayMap.getTime(remAddr)
				if !ok {
					globalDelayMap.writeTime(remAddr, time.Now())
					return
				}
				if time.Since(t) < 5*time.Second {
					globalDelayMap.writeTime(remAddr, time.Now()) // рестартуем таймер
					return
				}
				globalDelayMap.delRecord(remAddr)
			}

			// получаем весь массив сотрудников, который будем возвращать, т.к. параметр обрабатывается как like
			jsonSliceOfUsersByName := GetUsersByNameLightVersionAttributes(ins, userName)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write(jsonSliceOfUsersByName)
			return
		} else {
			http.NotFound(w, r)
			return
		}
	}
}

//*******************************************
// блок "Подразделения"
// получить все подразделения из 1С:ЗУП. ping - метод.
func RestGetFromZupPingAllDepartaments() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// create request
		req, err := http.NewRequest(http.MethodGet, config.GetApi1CZupAllDepartaments(), nil)
		if err != nil {
			log.Error("rest handlers.RestGetFromZupPingAllDepartaments http.NewRequest error: ", err)
		}
		// create client, do request and get response
		client := http.Client{Timeout: client1CTimeout * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			strError := fmt.Sprintf("rest: handlers.RestGetFromZupPingAllDepartaments client.Do(req) error: %v", err)
			log.Error(strError)
			w.WriteHeader(http.StatusNotFound) //StatusRequestTimeout
			w.Write([]byte("1C:ZUP is not available: " + strError))
			return
		}
		defer resp.Body.Close()

		// have got body of response
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Error("rest handlers.RestGetFromZupPingAllDepartaments io.ReadAll(resp.Body) error: %v", err)
		}

		HandleZupPingAllDepartaments(w, body)
	}
}

// Testing-Debug. Запишем в БД одно подразделение из запроса (из Postman-а)
func RestHandleDebugWriteSingleDepartament(ins *repository.PostgreInstance) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			log.Error("handlers.RestHandleDebugWriteSingleDepartament: io.ReadAll(Body): %v", err)
			http.Error(w, "500 - Something bad happened!", 500)
			return
		}
		defer r.Body.Close()

		strAnswer, err := handleAllDepartamentForCRUD(ins, body)
		if err != nil {
			log.Error("handlers.RestHandleDebugWriteSingleDepartament handle bodi error: %v", err)
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(err.Error()))
			return
		}
		log.Info(strAnswer)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("200 - ok!"))
		// must return "w" - response ("ok", "200" or not)
	}
}

// получить все подразделения из 1С:ЗУП и записать их в базу MD
func RestHandleFromZupAllDepartament(ins *repository.PostgreInstance) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		// create request
		req, err := http.NewRequest(http.MethodGet, config.GetApi1CZupAllDepartaments(), nil)
		if err != nil {
			log.Error("rest handlers.RestHandleFromZupAllDepartament http.NewRequest error: %v", err)
		}
		// create client, do request and get response
		client := http.Client{Timeout: client1CTimeout * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			log.Error("rest handlers.RestHandleFromZupAllDepartament client.Do(req) error: %v", err)
			strError := fmt.Sprintf("rest: handlers.RestHandleFromZupAllDepartament client.Do(req) error: %v", err)
			log.Error(strError)
			rw.WriteHeader(http.StatusNotFound) //StatusRequestTimeout
			rw.Write([]byte("1C:ZUP is not available: " + strError))
			return
		}
		defer resp.Body.Close()

		// have got body of response
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Error("rest handlers.RestHandleFromZupAllDepartament io.ReadAll(resp.Body) error: %v", err)
		}

		// собственно обработчик:
		strAnswer, err := handleAllDepartamentForCRUD(ins, body)
		if err != nil {
			log.Error("rest handlers.RestHandleFromZupAllDepartament handleAllDepartamentForCRUD error: %v", err)
		}
		log.Info("handlers.RestHandleFromZupAllDepartament answer: %s", strAnswer)
	}
}

//*******************************************
// блок "Рассылка уведомлений о днях рождения"

//------------------------------------------------------------
// установить пары observer - bd_owner
func RestSetOOCoupleForBdNotifications(ins *repository.PostgreInstance) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			rw.WriteHeader(http.StatusMethodNotAllowed)
			rw.Write([]byte("Need POST method. Please try one more time."))
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			log.Error("handlers.RestSetOOCoupleForBdNotifications: io.ReadAll(Body) %v", err)
			http.Error(rw, "500 - Something bad happened!", 500)
			return
		}
		defer r.Body.Close()

		strAnswer, err := handleSetOOCoupleForBdNotifications(ins, body)
		if err != nil {
			log.Error("handlers.handleSetOOCoupleForBdNotifications handle bodi error: %v", err)
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(err.Error()))
			return
		}
		log.Info(strAnswer)
		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte("200 - ok! " + strAnswer))
	}
}

//------------------------------------------------------------
// запустить поиск и рассылку
func RestSendBdNotifications(ins *repository.PostgreInstance) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		// собственно обработчик:
		sendBdNotifications(ins)
		status := "handlers.RestSendBdNotifications : Check birthdays for notifications is on"
		log.Info(status)
		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte(status + ", статусы смотрите в логах"))
	}
}

//------------------------------------------------------------
