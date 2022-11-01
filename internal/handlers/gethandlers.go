package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"strconv"
	"strings"
	"sync"

	config "mdata/configs"
	dom "mdata/internal/domain"
	"mdata/internal/repository"
	log "mdata/pkg/logging"
)

var insLdapConn *repository.LdapConn

func openLdapConn() error {
	cfg := config.GetCfg()
	// cfg := &dom.Config{}
	// cfg.ADHost = repository.ReadConfig("ad.host")
	// cfg.ADUsername = repository.ReadConfig("ad.username")
	// cfg.ADPassword = repository.ReadConfig("ad.password")
	// cfg.ADPort = repository.ReadConfig("ad.port")

	conn, err := repository.LdapSetConnection(cfg)
	if err != nil {
		return fmt.Errorf("handlers.openLdapConn error: %v", err)
	}
	insLdapConn = &repository.LdapConn{Conn: conn}
	return nil
}

func PingMD() string {
	return "Hello, I'm really MD!"
}

func ZupPing(Body io.ReadCloser) (string, error) {

	var body []byte
	body, err := ioutil.ReadAll(Body)
	if err != nil {
		return "", fmt.Errorf("handlers.PingZup: ioutil.ReadAll(Body): %v", err)
	}
	retstr := string(body)
	return retstr, nil
}

// Функция для записи массива сотрудников базу. Здесь считываем ВСЕХ user-ов из 1С:ЗУП и сравниваем их с базой - ищем и обрабатываем изменения
func handleZupWriteAllUsers(ins *repository.PostgreInstance, data []byte) (string, error) {

	// Откроем ldap соединение:
	if (insLdapConn == nil) || (insLdapConn.Conn.IsClosing()) {
		err := openLdapConn()
		if err != nil {
			return "handlers.HandleZupWriteAllUsers error: ", err
		}
	}
	defer insLdapConn.Conn.Close()

	var gettingUsers dom.AGUsers

	err := json.Unmarshal(data, &gettingUsers)
	if err != nil {
		return "handlers.HandleZupWriteAllUsers unmarshal gettingUsers error: ", err
	}

	for i, v := range gettingUsers.Users {
		if i%700 == 0 {
			log.Info("i = %d", i)
		}
		{
			// каждого user_а отрабатываем отдельно:
			// TODO: переписать с использованием goroutine
			currEmail, err := getUserEmailByEmployeeTabNo(insLdapConn, v)
			if err != nil {
				return "", fmt.Errorf("handlers.HandleZupWriteSingleUser getting email error: %v, for user %v, with userID %v", err, v.UserName, v.UserID)
			}

			_, err = handleSingleUserForCRUD(ins, currEmail, &v)
			if err != nil {
				log.Error("handlers.HandleZupWriteAllUsers error: ", err)
				strAnswer := "Ошибка при проверке к загрузке users"
				return strAnswer, fmt.Errorf("handlers.HandleZupWriteAllUsers error: %v", err)
			}
		}
	}
	strAnswer := fmt.Sprintf("Проверено к загрузке %s users", strconv.Itoa(len(gettingUsers.Users)))
	return strAnswer, nil
}

func getUserEmailByEmployeeTabNo(insLdapConn *repository.LdapConn, user dom.User) (string, error) {

	var resEmail string
	var wg sync.WaitGroup

	fn := func() func(string) {
		return func(searchParam string) {
			defer func() {
				wg.Done()
			}()
			curEmail, err := insLdapConn.GetUserEmailByID(searchParam)
			if err != nil {
				log.Error("handlers.getUserEmailByEmployeeTabNo getting email error: %v, for user %v, with userID %v and searchParam %v", err, user.UserName, user.UserID, searchParam)
				return
			}
			if curEmail != "" {
				resEmail = curEmail
			}
		}
	}()

	for _, empl := range user.Employees {
		wg.Add(1)
		go fn(strings.TrimSpace(user.UserID)) // поиск по коду физ. лица
		wg.Add(1)
		go fn(empl.EmployeeId) // поиск по коду сотрудника
		// wg.Add(1)
		// go fn(ctx, empl.EmpTabNumber) // поиск по "личному номеру"

		if resEmail != "" { // когда несколько Employees у user - а, и resEmail уже нашли
			break
		}
	}
	wg.Wait()

	return resEmail, nil
}
