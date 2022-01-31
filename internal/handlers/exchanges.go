package handlers

import (
	"encoding/json"
	dom "mdata/internal/domain"
	log "mdata/internal/logging"
	"mdata/internal/repository"
	"sync"
)

func registerToExchange(ins *repository.Instance, exch *dom.ExchangeStruct) error {
	_, err := ins.AddToExchange(exch)
	if err != nil {
		return err
	}
	return nil
}

func getUsersArrayWithEmailToPUTQuery(ins *repository.Instance, reasonID int) ([]byte, *map[int]int, []dom.User, error) {

	var sliceOfByte []byte
	ptrEmptyExchMap := &map[int]int{} // в мапе получаем id отправляемой строки и attempt_count

	usersToExchangeSlice, ptrExchMap, err := ins.GetAllUsersToExchange(reasonID)
	if err != nil {
		log.Error("getUsersArrayWithEmailToPUTQuery", err)
		return sliceOfByte, ptrEmptyExchMap, usersToExchangeSlice, err // пустой
	}
	if len(usersToExchangeSlice) == 0 {
		return sliceOfByte, ptrEmptyExchMap, usersToExchangeSlice, nil // пустой
	}

	usersToExchangeArray := dom.AGUsers{}
	usersToExchangeArray.Users = usersToExchangeSlice

	sliceOfByte, err = json.Marshal(usersToExchangeArray)
	if err != nil {
		log.Error("marshal usersToExchangeArray error", err)
		return sliceOfByte, ptrEmptyExchMap, usersToExchangeSlice, err // пустой
	}

	return sliceOfByte, ptrExchMap, usersToExchangeSlice, nil
}

func putRequestFromZupLoading(ins *repository.Instance, exMap *map[int]int, errorStr string) {
	var wg sync.WaitGroup
	// для записи логов будем возвращать из горутин не только ошибку, но и id записи, куда пытались записать
	type resultStruct struct {
		k   int
		err error
	}
	// semafore patern
	errChan := make(chan resultStruct, len(*exMap))
	for k, v := range *exMap { // в мапе (exMap) получаем id отправленной строки и attempt_count для понимания, куда (в какую строку) записывать ошибку или 'ok'
		wg.Add(1)
		k, v, errorStr := k, v, errorStr
		go func() {
			var curResultStruct resultStruct
			defer func() {
				wg.Done()
				errChan <- curResultStruct
			}()
			err := ins.SetExchangeStatus(k, v, errorStr)
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
			log.Error("handlers.putRequestFromZupLoading error: при попытке записи в БД статуса 'PUT to 1C' для ex_id = %v получена ошибка: %v ", k, err)
		} else {
			log.Info("handlers.putRequestFromZupLoading: статус попытки записи 'PUT to 1C' успешно записан в БД для ex_id = %v", k)
		}
	}
}
