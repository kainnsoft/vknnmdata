package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"strconv"
	"strings"
	"sync"

	dom "mdata/internal/domain"
	log "mdata/internal/logging"
	"mdata/internal/repository"
)

var insLdapConn *repository.LdapConn

func openLdapConn() error {
	cfg := &dom.Config{}
	cfg.ADHost = repository.ReadConfig("ad.host")
	cfg.ADUsername = repository.ReadConfig("ad.username")
	cfg.ADPassword = repository.ReadConfig("ad.password")
	cfg.ADPort = repository.ReadConfig("ad.port")

	conn, err := repository.LdapSetConnection(cfg)
	if err != nil {
		return fmt.Errorf("handlers.openLdapConn error: %v", err)
	}
	insLdapConn = &repository.LdapConn{Conn: conn}
	return nil
}

func PingMD() string {
	return "Hello, I'm really MD!!!"
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
func handleZupWriteAllUsers(ins *repository.Instance, data []byte) (string, error) {

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

func getUserEmailByEmployeeTabNo_old(insLdapConn *repository.LdapConn, user dom.User) (string, error) {

	var curEmail string
	var err error
	for _, empl := range user.Employees { ///  TODO - всё переписать: искать все три case - а параллельно
		curEmail, err = insLdapConn.GetUserEmailByID(strings.TrimSpace(empl.EmployeeId)) // поиск по коду сотрудника
		if err != nil {
			return "", fmt.Errorf("handlers.getUserEmailByEmployeeTabNo getting email error: %v, for user %v, with userID %v and Empl.TabNumber %s", err, user.UserName, user.UserID, empl.EmpTabNumber)
		}
		if curEmail != "" {
			break
		}
		curEmail, err = insLdapConn.GetUserEmailByID(strings.TrimSpace(user.UserID)) // поиск по коду физ. лица
		if err != nil {
			return "", fmt.Errorf("handlers.getUserEmailByEmployeeTabNo getting email error: %v, for user %v, with userID %v and Empl.TabNumber %s", err, user.UserName, user.UserID, empl.EmpTabNumber)
		}
		if curEmail != "" {
			break
		}
		curEmail, err = insLdapConn.GetUserEmailByID(strings.TrimSpace(empl.EmpTabNumber)) // поиск по "личному номеру"
		if err != nil {
			return "", fmt.Errorf("handlers.getUserEmailByEmployeeTabNo getting email error: %v, for user %v, with userID %v and Empl.TabNumber %s", err, user.UserName, user.UserID, empl.EmpTabNumber)
		}
		if curEmail != "" {
			break
		}
	}
	return curEmail, nil
}

func getUserEmailByEmployeeTabNo(insLdapConn *repository.LdapConn, user dom.User) (string, error) {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var resEmail string
	//getEmail := make(chan string, 5)

	fn := func() func(context.Context, string) {
		return func(ctx context.Context, searchParam string) {
			//for {
			select {
			case <-ctx.Done():
				//log.Info("I'm canceled! (searchParam = %s)", searchParam)
				return
			default:
				curEmail, err := insLdapConn.GetUserEmailByID(searchParam)
				if err != nil {
					log.Error("handlers.getUserEmailByEmployeeTabNo getting email error: %v, for user %v, with userID %v", err, user.UserName, user.UserID)
				}
				// if curEmail == "" {
				// 	time.Sleep(5 * time.Second)
				// }
				if curEmail != "" {
					//getEmail <- curEmail
					resEmail = curEmail
				}
				//log.Info("Param = %s, curEmail = %s", searchParam, curEmail)
			}
			//}
		}
	}()

	var wg sync.WaitGroup
	for _, empl := range user.Employees {
		wg.Add(1)
		go func(employeeId string) {
			defer wg.Done()
			fn(ctx, employeeId) // поиск по коду сотрудника
		}(empl.EmployeeId)
		wg.Add(1)
		go func(userID string) {
			defer wg.Done()
			fn(ctx, strings.TrimSpace(userID)) // поиск по коду физ. лица
		}(user.UserID)
		wg.Add(1)
		go func(emplTabNumber string) {
			defer wg.Done()
			fn(ctx, emplTabNumber) // поиск по "личному номеру"
		}(empl.EmpTabNumber)
	}

	//resEmail := <-getEmail
	//cancel()

	wg.Wait()
	//log.Info("main = %s", resEmail)

	return resEmail, nil
}

func getUserEmailByUserID(user dom.User) (string, error) {
	var curEmail string
	var err error
	curEmail, err = insLdapConn.GetUserEmailByID(strings.TrimSpace(user.UserID))
	if err != nil {
		return "", fmt.Errorf("handlers.getUserEmailByUserID getting email error: %v, for user %v, with userID %v", err, user.UserName, user.UserID)
	}
	return curEmail, nil
}
