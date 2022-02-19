package handlers

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	dom "mdata/internal/domain"
	"mdata/internal/repository"
	log "mdata/pkg/logging"
)

// Функция для записи одного user-а в базу, если не нашли такого по GUID, либо обновление, если нашли:
func handleSingleUserForCRUD(ins *repository.Instance, currEmail string, usr *dom.User) (string, error) {

	var str string
	var err error

	selectedUserCRUDStatus, err := checkSingleUserForCRUD(ins, usr, currEmail)
	switch selectedUserCRUDStatus {
	case -1:
		// error
		return str, err
	case 0: // не нашли (новый user), будем добавлять
		str, err = addUser(ins, currEmail, usr)
		if err != nil {
			return str, err
		}
		// если добавляем user-а с email-ом (только в этом случае), зрегистрируем его к обмену с 1С:ЗУП
		if strings.TrimSpace(currEmail) != "" {
			usr.UserEmail = currEmail
			if err != nil {
				log.Error("handleSingleUserForCRUD addUser, marshal error", err)
				log.Error("Не зарегистрирован к обмену добавленный user с email: фио - %s, таб. № %s", usr.UserName, usr.UserID)
				return str, err
			}
			exch := dom.ExchangeStruct{BaseID: 2, ReasonID: 1, RowData: usr.UserGUID} // ReasonID: 1 - это send emails to 1CZUP (PUT request)
			err = registerToExchange(ins, &exch)
			if err != nil {
				log.Error("Не зарегистрирован к обмену добавленный user с email: фио - %s, таб. № %s", usr.UserName, usr.UserID)
				return str, err
			}

			// также, если это новый user и с email-ом, то скорее всего ему понадобится доступ в 1С. Зарегистрируем его к обмену с базой 1С, создающей пользователей 1С
			go func() {
				exch1С := dom.ExchangeStruct{BaseID: 6, ReasonID: 2, RowData: usr.UserGUID} // ReasonID: 2 - это send hired-employees to 1CService base to give permissions (PUT request)
				err = registerToExchange(ins, &exch1С)
				if err != nil {
					log.Error("Не зарегистрирован к обмену новый user для создания пользователя в 1С: фио - %s, таб. № %s", usr.UserName, usr.UserID)
				}
			}()
		}

		// check for update employees below
	case 1:
		// нашли (ничего с user-ом не делаем)
		// check for update employees below
	case 2: // need for update
		str, err = updateUser(ins, currEmail, usr)
		if err != nil {
			return str, err
		}
	case 22: // need for update email. Обработаем этот case отдельно, т.к. нужно зарегистрировать user-а к обмену с 1С:ЗУП
		str, err = updateUser(ins, currEmail, usr)
		if err != nil {
			return str, err
		}
		// т.к. нужно обновить email, запишем это в файлик, который в пятницу уйдет в бухгалтерию
		if strings.TrimSpace(currEmail) == "" {
			log.InfoBuch("Удалён email у сотрудника: %v c таб. № %v", usr.UserName, usr.UserID)
			log.InfoBuchFull("Удалён email у сотрудника: %v c таб. № %v", usr.UserName, usr.UserID)
		} else {
			log.InfoBuch("Установлен email: %v для сотрудника: %v c таб. № %v", currEmail, usr.UserName, usr.UserID)
			log.InfoBuchFull("Установлен email: %v для сотрудника: %v c таб. № %v", currEmail, usr.UserName, usr.UserID)

			// также, если у user-а появился email, то скорее всего ему понадобится доступ в 1С. Зарегистрируем его к обмену с базой 1С, создающей пользователей 1С
			go func() {
				exch1С := dom.ExchangeStruct{BaseID: 6, ReasonID: 2, RowData: usr.UserGUID} // ReasonID: 2 - это send hired-employees to 1CService base to give permissions (PUT request)
				err = registerToExchange(ins, &exch1С)
				if err != nil {
					log.Error("Не зарегистрирован к обмену новый user для создания пользователя в 1С: фио - %s, таб. № %s", usr.UserName, usr.UserID)
				}
			}()

		}
		// если обновляем user-а с email-ом, зрегистрируем его к обмену с 1С:ЗУП
		usr.UserEmail = currEmail
		go func() {
			exch := dom.ExchangeStruct{BaseID: 2, ReasonID: 1, RowData: usr.UserGUID} //string(sliceOfByte)}
			err = registerToExchange(ins, &exch)                                      // ReasonID: 1 - это send email to 1CZUP
			if err != nil {
				log.Error("Не зарегистрирован к обмену с 1С:ЗУП(email) обновленный user: фио - %s, таб. № %s по причине %v", usr.UserName, usr.UserID, err)
			}
		}()
		// check for update employees below
	}

	// check for update employees:
	// добавим/обновим сразу всех сотрудников пользователя (по совместительству и т.д.) с подразделениями, должностями и т.д.
	// добавление через стандартную процедуру "check for update" (чтобы не плодить методы - из-за пары проверок на одной операции добавления производительность сильно не упадет; таких операций в сутки не больше пяти...)
	err = handleAllUserEmployeesForCRUD(ins, usr)
	if err != nil {
		log.Error("handlers.handleSingleUserForCRUD creating Employee error: %v, for user %v, with userID %v", err, usr.UserName, usr.UserID)
		return str, fmt.Errorf("handlers.handleSingleUserForCRUD creating Employee error: %v, for user %v, with userID %v", err, usr.UserName, usr.UserID)
	}

	return str, nil
}

// проверим, нужно ли обновлять пользователя исходя из записи о нем в DB
func checkSingleUserForCRUD(ins *repository.Instance, usr *dom.User, currEmail string) (int, error) {

	// найдем пользователя в DB
	gettingUser, err := ins.SelectUserByGUID(usr.UserGUID)
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			return 0, err // не нашли, нужно добавлять
		}
		return -1, err // ошибка при поиске
	}

	// проверим, нужно ли обновить аттрибуты
	needUpdate := false
	if gettingUser.UserName != usr.UserName {
		log.Info("NUpdt UserName for user %s  with Id %s  в базе: %s  загружаем: %s", usr.UserName, usr.UserID, gettingUser.UserName, usr.UserName)
		needUpdate = true
	}
	if strings.TrimSpace(gettingUser.UserID) != strings.TrimSpace(usr.UserID) {
		log.Info("NUpdt UserID for user %s  with Id %s  в базе: %s  загружаем: %s", usr.UserName, usr.UserID, gettingUser.UserID, usr.UserID)
		needUpdate = true
	}
	curUserBirthday := gettingUser.UserBirthday.DateToString()
	vUserBirthday := usr.UserBirthday.DateToString()
	if curUserBirthday != vUserBirthday {
		log.Info("NUpdt UserBirthday for user %s  with Id %s  в базе: %s  загружаем: %s", usr.UserName, usr.UserID, gettingUser.UserBirthday, usr.UserBirthday)
		needUpdate = true
	}
	if gettingUser.UserEmail != currEmail {
		log.Info("NUpdt UserEmail for user %v with Id %v в базе: %v загружаем: %v", usr.UserName, usr.UserID, gettingUser.UserEmail, currEmail)
		// обрабатываем этот case отдельно
		return 22, nil
	}
	if needUpdate {
		return 2, nil
	}
	return 1, nil // нашли, не нужно обновлять пользователя
}

func addUser(ins *repository.Instance, currEmail string, usr *dom.User) (string, error) {
	str, err := ins.AddUserToDB(usr, currEmail)
	if err != nil {
		return str, err
	}
	log.Info("added user %v, %v, %v, %v, %v, %v", usr.UserGUID, usr.UserName, usr.UserID, usr.UserBirthday, currEmail, str)

	return str, err
}

func updateUser(ins *repository.Instance, currEmail string, usr *dom.User) (string, error) {
	str, err := ins.UpdateUserInDB(usr, currEmail)
	if err != nil {
		return str, err
	}
	log.Info("updated user %v, %v, %v, %v, %v, %v", usr.UserGUID, usr.UserName, usr.UserID, usr.UserBirthday, currEmail, str)
	return str, err
}

//------------------------------------------------------------
// все аттрибуты
// Отдаем данные всех user-ов, сотрудники которых работают (актуальны)  (все аттрибуты)
func GetActEmployeesAllAttributes(ins *repository.Instance) ([]byte, error) {
	// пока ВСЕ (не один) из базы:
	allEmployeesData, err := ins.GetAllActualUsersAllAttributes()
	if err != nil {
		log.Error("GetActEmployeesAllAttributes", err)
	}

	allEmployees := dom.AGUsers{}
	allEmployees.Users = allEmployeesData

	sliceOfByte, err := json.MarshalIndent(allEmployees, "", "  ")
	if err != nil {
		log.Error("marshal GetActEmployeesAllAttributes error", err)
	}

	return sliceOfByte, nil
}

// Отдаем данные всех user-ов, сотрудники которых уволены  (все аттрибуты)
func GetFiredEmployeesUsersAllAttributes(ins *repository.Instance, dateFrom time.Time) ([]byte, error) {
	// пока ВСЕ (не один) из базы:
	allEmployeesData, err := ins.GetUsersFiredFrom(dateFrom)
	if err != nil {
		log.Error("GetFiredEmployeesUsersAllAttributes", err)
	}

	allEmployees := dom.AGUsers{}
	allEmployees.Users = allEmployeesData

	sliceOfByte, err := json.MarshalIndent(allEmployees, "", "  ")
	if err != nil {
		log.Error("marshal GetFiredEmployeesUsersAllAttributes error", err)
	}

	return sliceOfByte, nil
}

// вернём массив только работающих пользователей с подходящими кодами - табельными номерами (?tabno=8337) все атрибуты:
func GetUsersByTabNoAllAttributes(ins *repository.Instance, tabno string) []byte {
	// пока ВСЕ (не один) из базы:
	usersByTabNoSlice, err := ins.GetActualUsersByTabNoAllAttributes(tabno)
	if err != nil {
		log.Error("GetUsersByTabNoAllAttributes", err)
	}

	usersByTabNo := dom.AGUsers{}
	usersByTabNo.Users = usersByTabNoSlice

	sliceOfByte, err := json.MarshalIndent(usersByTabNo, "", "  ")
	if err != nil {
		log.Error("marshal GetUsersByTabNoAllAttributes error", err)
	}

	return sliceOfByte
}

// вернём массив только работающих пользователей с подходящими ФИО (?name=захаро) все атрибуты:
func GetUsersByNameAllAttributes(ins *repository.Instance, userName string) []byte {
	// пока ВСЕ (не один) из базы:
	usersByUserNameSlice, err := ins.GetActualUsersByUserNameAllAttributes(userName)
	if err != nil {
		log.Error("GetUsersByNameAllAttributes", err)
	}

	usersByUserName := dom.AGUsers{}
	usersByUserName.Users = usersByUserNameSlice

	sliceOfByte, err := json.MarshalIndent(usersByUserName, "", "  ")
	if err != nil {
		log.Error("marshal GetUsersByNameAllAttributes error", err)
	}

	return sliceOfByte
}

//------------------------------------------------------------
// облегчённые аттрибуты
// вернём массив только работающих пользователей с подходящими кодами - табельными номерами (?tabno=8337) облегчённые атрибуты:
func GetUsersByTabNoLightVersionAttributes(ins *repository.Instance, tabno string) []byte {
	// пока ВСЕ (не один) из базы:
	usersByTabNoSlice, err := ins.GetActualUsersByTabNoLightVersionAttributes(tabno)
	if err != nil {
		log.Error("GetUsersByTabNoLightVersionAttributes", err)
	}

	usersByTabNo := dom.AGUsers{}
	usersByTabNo.Users = usersByTabNoSlice

	sliceOfByte, err := json.MarshalIndent(usersByTabNo, "", "  ")
	if err != nil {
		log.Error("marshal GetUsersByTabNoLightVersionAttributes error", err)
	}

	return sliceOfByte
}

// вернём массив только работающих пользователей с подходящими ФИО (?name=захаро) облегчённые атрибуты:
func GetUsersByNameLightVersionAttributes(ins *repository.Instance, userName string) []byte {
	// пока ВСЕ (не один) из базы:
	usersByUserNameSlice, err := ins.GetActualUsersByUserNameLightVersionAttributes(userName)
	if err != nil {
		log.Error("GetUsersByNameLightVersionAttributes", err)
	}

	usersByUserName := dom.AGUsers{}
	usersByUserName.Users = usersByUserNameSlice

	sliceOfByte, err := json.MarshalIndent(usersByUserName, "", "  ")
	if err != nil {
		log.Error("marshal GetUsersByNameLightVersionAttributes error", err)
	}

	return sliceOfByte
}
