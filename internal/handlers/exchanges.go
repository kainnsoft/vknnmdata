package handlers

import (
	"encoding/json"
	"fmt"
	dom "mdata/internal/domain"
	"mdata/internal/repository"
	"mdata/internal/utils"
	log "mdata/pkg/logging"
	"sync"
)

// производит запись в таблицу exchanges, когда произошли изменения (например у user-а появился email)
func registerToExchange(ins *repository.Instance, exch *dom.ExchangeStruct) error {
	_, err := ins.AddToExchange(exch)
	if err != nil {
		return err
	}
	return nil
}

// возвращаем массив user-ов, мапу с id отправляемой строки и attempt_count, массив user-ов для формирования письма, ошибку
// TODO перенести маршаллинг массива user-ов в sliceOfByte в вызывающую функцию
func getUsersArrayWithEmailToPUTQuery(ins *repository.Instance, reasonID int) ([]byte, *map[int]dom.Exchange1СErrorsStruct, []dom.User, error) {

	var sliceOfByte []byte
	ptrEmptyExchMap := &map[int]dom.Exchange1СErrorsStruct{} // в мапе получаем id отправляемой строки и attempt_count

	// usersToExchangeSlice - все полные данные о user-е
	// ptrExchMap - № строки таблицы exchanges + {userGuid, attemptCount} - в нее будем принимать ответ от 1С
	usersToExchangeSlice := make([]dom.User, 0)
	// вначале получим список guid-ов нужных user-ов
	userGuidExchangeSlice, ptrExchMap, err := ins.GetAllUsersToExchange(reasonID)
	if err != nil {
		log.Error("handlers.getUsersArrayWithEmailToPUTQuery: ошибка при получении списка guid-ов: ", err)
		return sliceOfByte, ptrEmptyExchMap, usersToExchangeSlice, err // пустой
	}
	// теперь получим список user-ов со всеми атрибутами по их guid-ам
	usersToExchangeSlice, err = ins.GetCastomUserListAllAttributes(userGuidExchangeSlice)
	if err != nil {
		log.Error("handlers.getUsersArrayWithEmailToPUTQuery: ошибка при получении списка user-ов:", err)
		return sliceOfByte, ptrEmptyExchMap, usersToExchangeSlice, err // пустой
	}
	if len(usersToExchangeSlice) == 0 {
		return sliceOfByte, ptrEmptyExchMap, usersToExchangeSlice, nil // пустой
	}

	usersToExchangeArray := dom.AGUsers{}
	usersToExchangeArray.Users = usersToExchangeSlice

	sliceOfByte, err = json.Marshal(usersToExchangeArray)
	if err != nil {
		log.Error("handlers.getUsersArrayWithEmailToPUTQuery: marshal usersToExchangeArray error", err)
		return sliceOfByte, ptrEmptyExchMap, usersToExchangeSlice, err // пустой
	}

	return sliceOfByte, ptrExchMap, usersToExchangeSlice, nil
}

// функция подготавливает запись в БД (таблицу "exchanges") результат выгрузки в 1С:CreateUser и 1С:ЗУП (о записанных email)
// если до 1С:CreateUser не достучались (body пустой), то во все выгруженные строки записываем ошибку и инкрементируем attempt_count
// если из 1С:CreateUser получен не пустой body, то читаем его - какие-то строки будут "Success", а какие-то с ошибкой
func putRequestFrom1CLoadingPrepare(ins *repository.Instance, body []byte, exMap *map[int]dom.Exchange1СErrorsStruct, errorStr string) {
	var mapFrom1C map[string]string = map[string]string{}
	if len(body) > 0 {
		// получаемая из 1С структура ответа (описание здесь -
		// http://wiki.vodokanal-nn.ru/bin/view/ОПП%20-%20Инструкции%20для%20разработчиков/Не%201С/Master%20Data/Интеграции/MD%20--%3E%201С%3ACreateUser/Автоматическое%20создание%20пользователей/):
		var mapArrFrom1C map[string][]dom.Response1CUserStatusStruct = map[string][]dom.Response1CUserStatusStruct{}

		err := json.Unmarshal(body, &mapArrFrom1C)
		if err != nil {
			errStr := fmt.Sprintf("unmarshal putRequestFrom1CLoadingSwitch error: не удалось разобрать строку, полученную из 1С, по причине: %v", err)
			log.Error(errStr)
			repository.SendEmailTo1CAdmins(ins, errStr) // критичная ошибка, нужно отслеживать - отправляем администратору
			return
		}
		sliceFrom1C, ok := mapArrFrom1C[repository.Array1Cname]
		if !ok {
			errStr := "putRequestFrom1CLoadingSwitch error: не удалось разобрать map-у, полученную из 1С, по причине: не найден ключ UsersStatus"
			log.Error(errStr)
			repository.SendEmailTo1CAdmins(ins, errStr) // критичная ошибка, нужно отслеживать - отправляем администратору
			return
		}

		// для O1 в дальнейшей обработке пересоберём slice в мапу
		for _, v := range sliceFrom1C {
			if _, ok := mapFrom1C[v.UserGuid]; !ok {
				mapFrom1C[v.UserGuid] = v.Status
			}
		}
		if len(mapFrom1C) > 0 {
			go utils.SendEmailTo1CAdminsRespCode200(ins, mapFrom1C)
		}
	}

	// теперь запишем подготовленные данные (mapFrom1C) в БД (в таблицу "exchanges")
	putRequestFrom1CLoading(ins, mapFrom1C, exMap, errorStr)
}

// Действия функции зависят от параметра mapFrom1C:
// - кодга нет body, т.е. ответ от 1С:CreateUser или 1С:ЗУП не 200, а ошибка. Запишем одну и ту же ошибку во все строки,
// - а когда от 1С:CreateUser или 1С:ЗУП пришёл ответ 200, а ошибка может быть влюбой из строк -
//   будем перебирать exMap и по UserGuid искать соответствующую запись в mapFrom1C. Если нашли, то читаем ответ из 1С, а если нет, то пишем неопознанную ошибку
func putRequestFrom1CLoading(ins *repository.Instance, mapFrom1C map[string]string, exMap *map[int]dom.Exchange1СErrorsStruct, errorStr string) {
	var wg sync.WaitGroup
	// для записи логов будем возвращать из горутин не только ошибку, но и id записи, куда пытались записать
	type resultStruct struct {
		k   int
		err error
	}
	// semafore pattern
	errChan := make(chan resultStruct, len(*exMap))
	for k, v := range *exMap { // в мапе (exMap) получаем id отправленной строки и attempt_count для понимания, куда (в какую строку) записывать ошибку или 'ok'
		wg.Add(1)
		//--------------------------
		curStatus := mapFrom1C[v.UserGUID] // из 1С-ной мапы получим статус по guid-у user-а. Если не нашлость, то curStatus будет = ""
		if len(curStatus) > 0 {
			if curStatus == repository.StatusSuccess {
				errorStr = repository.Response200ok
			} else {
				errorStr = curStatus
			}
		}
		//--------------------------
		k, v, errorStr := k, v, errorStr
		go func() {
			var curResultStruct resultStruct
			defer func() {
				wg.Done()
				errChan <- curResultStruct
			}()
			if len(errorStr) > 300 { // обрезаем ошибку, чтобы поместилась в строку БД
				errorStr = errorStr[0:299]
			}
			err := ins.SetExchangeStatus(k, v.AttemptCount, errorStr)
			curResultStruct = resultStruct{k: k, err: err}
		}()
	}
	wg.Wait() // чтобы гарантированно зайти в for ниже
	// TODO - пусть горутины пишут в файл
	for len(errChan) > 0 {
		resStruct := <-errChan
		k := resStruct.k
		err := resStruct.err
		if err != nil {
			log.Error("handlers.putRequestFrom1CLoading error: при попытке записи в БД статуса 'PUT to 1C' для ex_id = %v получена ошибка: %v ", k, err)
		} else {
			log.Info("handlers.putRequestFrom1CLoading: статус попытки записи 'PUT to 1C' успешно записан в БД для ex_id = %v", k)
		}
	}
}
