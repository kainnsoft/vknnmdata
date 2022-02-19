package handlers

import (
	"encoding/json"
	"fmt"
	"mdata/internal/domain"
	"mdata/internal/repository"
	"mdata/internal/utils"
	log "mdata/pkg/logging"
	"strconv"
	"sync"
	"time"
)

//********************
const typeTomorrow string = "Завтра день рождения у: "
const typeIn3Days string = "Через три дня день рождения у: "
const typeNextWeek string = "На следующей неделе день рождения у: "
const typeNextMonth string = "В следующем месяце день рождения у: "

//********************
//  KEY - recipient - получатель рассылки, VALUE - slice map - (key: тип ("завтра", "след. месяц и т.д."), value: слайс bd_owners)
type sendList struct {
	mapRecipientPeriodsOwners map[string]map[string][]domain.User
	mu                        sync.RWMutex
}

func newSendList(initMapRPO map[string]map[string][]domain.User) *sendList {
	if initMapRPO != nil {
		return &sendList{
			mapRecipientPeriodsOwners: initMapRPO,
		}
	}
	return &sendList{
		mapRecipientPeriodsOwners: make(map[string]map[string][]domain.User),
	}
}

func (sl *sendList) GetValue(keyEmail string) map[string][]domain.User {
	sl.mu.RLock()
	defer sl.mu.RUnlock()
	val, ok := sl.mapRecipientPeriodsOwners[keyEmail]
	if ok {
		return val
	}
	return map[string][]domain.User{}

}

func (sl *sendList) SetValue(keyEmail string, value map[string][]domain.User) {
	sl.mu.Lock()
	defer sl.mu.Unlock()
	sl.mapRecipientPeriodsOwners[keyEmail] = value
}

//********************

// все рассылки делаем до наступления рабочего дня
func sendBdNotifications(ins *repository.Instance) {
	// создаём общий список рассылки. В него будем набивать адресатов и именинников в разрезе периодов
	commonSendList := newSendList(map[string]map[string][]domain.User{})
	var wg sync.WaitGroup
	// завтра
	wg.Add(1)
	go func() {
		defer wg.Done()
		getBdTomorrow(ins, commonSendList)
	}()

	// через три дня (за три дня до...)
	wg.Add(1)
	go func() {
		defer wg.Done()
		getBdInThreeDays(ins, commonSendList)
	}()

	// на следующей неделе (за три дня до...)
	wg.Add(1)
	go func() {
		defer wg.Done()
		getBdOnNextWeek(ins, commonSendList)
	}()

	// в следующем месяце (за три дня до...)
	wg.Add(1)
	go func() {
		defer wg.Done()
		getBdOnNextMonth(ins, commonSendList)
	}()

	wg.Wait()

	admins, err := ins.GetUserEmailsByNotificationsTypes(1)
	if err != nil {
		log.Error("bd_notifications handlers.sendBdNotifications GetBccAdmin error: %v", err)
	}
	bccAdmin := admins[0]
	if bccAdmin == "" {
		log.Error("bd_notifications handlers.sendBdNotifications GetBccAdmin error: no bccAdmin", err)
	}

	// цикл по мапе по адресатам
	for keyEmail, mapPeriodOwners := range *&commonSendList.mapRecipientPeriodsOwners {
		prepareSendLetterToSingleMail(bccAdmin, keyEmail, mapPeriodOwners)
	}
}

// Отправка писем:
func SendEmail(bccAdmin, recipient, body string) error {
	// адресаты
	recipients := []string{recipient}

	// тема
	subject := "Birthdays notification"
	// тело
	// отправка
	err := repository.SendMailToRecipient(recipients, bccAdmin, subject, body, "")
	if err != nil {
		return err
	}

	return nil
}

func getBdTomorrow(ins *repository.Instance, csl *sendList) { // каждый день в 8:00
	now := time.Now()
	tomorrow := now.AddDate(0, 0, 1)
	layout := "2006-01-02"
	templTomorrow := tomorrow.Format(layout)[4:10]
	_ = templTomorrow

	// mapBdOO map[string][]domain.User - одному observer-у (key) сопоставлено много bd_owner-ов (value)
	mapBdOO, err := ins.GetBdObserversOwnersByTempl(templTomorrow) // ins.GetBdOwnersOneDayDebug("-02-24") //
	if err != nil {
		log.Error("bd_notifications handlers.getBdTomorrow GetBdOwnersOneDay error: %v", err)
	}

	for keyEmail, bdOwnersSlice := range mapBdOO {
		currentMapPeriodOwners := csl.GetValue(keyEmail)
		currentMapPeriodOwners[typeTomorrow] = bdOwnersSlice
		csl.SetValue(keyEmail, currentMapPeriodOwners)
	}
}

func getBdInThreeDays(ins *repository.Instance, csl *sendList) { // каждый день в 8:00
	now := time.Now()
	in3Days := now.AddDate(0, 0, 3)
	layout := "2006-01-02"
	templIn3Days := in3Days.Format(layout)[4:10]

	// mapBdOO map[string][]domain.User - одному observer-у (key) сопоставлено много bd_owner-ов (value)
	mapBdOO, err := ins.GetBdObserversOwnersByTempl(templIn3Days)
	if err != nil {
		log.Error("bd_notifications handlers.getBdInThreeDays GetBdObserversOwnersByTempl error: %v", err)
	}

	for keyEmail, bdOwnersSlice := range mapBdOO {
		currentMapPeriodOwners := csl.GetValue(keyEmail)
		currentMapPeriodOwners[typeIn3Days] = bdOwnersSlice
		csl.SetValue(keyEmail, currentMapPeriodOwners)
	}
}

func getBdOnNextWeek(ins *repository.Instance, csl *sendList) {
	now := time.Now()
	if now.Weekday() == time.Friday { // по пятницам в 8:00
		nextWeek := utils.GetNextWeekTemplStruct(now)

		// mapBdOO map[string][]domain.User - одному observer-у (key) сопоставлено много bd_owner-ов (value)
		mapBdOO, err := ins.GetBdObserversOwnersnextWeek(nextWeek)
		if err != nil {
			log.Error("bd_notifications handlers.getBdOnNextWeek GetBdOwnersByTempl error: %v", err)
		}

		for keyEmail, bdOwnersSlice := range mapBdOO {
			currentMapPeriodOwners := csl.GetValue(keyEmail)
			currentMapPeriodOwners[typeNextWeek] = bdOwnersSlice

			csl.SetValue(keyEmail, currentMapPeriodOwners)
		}
	}
}

func getBdOnNextMonth(ins *repository.Instance, csl *sendList) {
	// TODO
	now := time.Now()
	firstOfNextMonth, _ := utils.NextMonth(now)
	threeDaysBefore := firstOfNextMonth.AddDate(0, 0, -4) // за четыре дня до наступления след. месяца в 8:00

	layout := "2006-01-02"
	nowFormat := now.Format(layout)
	threeDaysBeforeFormat := threeDaysBefore.Format(layout)
	if nowFormat == threeDaysBeforeFormat {
		templNextMonth := firstOfNextMonth.Format(layout)[4:8]
		// mapBdOO map[string][]domain.User - одному observer-у (key) сопоставлено много bd_owner-ов (value)
		mapBdOO, err := ins.GetBdObserversOwnersByTempl(templNextMonth)
		if err != nil {
			log.Error("bd_notifications handlers.getBdOnNextMonth GetBdOwnersByTempl error: %v", err)
		}

		for keyEmail, bdOwnersSlice := range mapBdOO {
			currentMapPeriodOwners := csl.GetValue(keyEmail)
			currentMapPeriodOwners[typeNextMonth] = bdOwnersSlice
			csl.SetValue(keyEmail, currentMapPeriodOwners)
		}
	}
}

// prepare letters //подготовка письма
func prepareSendLetterToSingleMail(bccAdmin, keyEmail string, mapPeriodOwners map[string][]domain.User) {
	var body string
	//fmt.Println("keyEmail ======= ", keyEmail)
	orderSlice := []string{typeTomorrow, typeIn3Days, typeNextWeek, typeNextMonth}
	for _, v := range orderSlice {
		if bdOwnersSlice, ok := mapPeriodOwners[v]; ok {
			body = body + "-------------------------- \n"
			body = body + v + "\n"
			for _, userData := range bdOwnersSlice {
				body = body + "        " + userData.UserName + " (" + userData.UserBirthday.Format("02-01-2006") + ")" + " \r\n"
			}
		}
	}
	SendEmail(bccAdmin, keyEmail, body)
}

// установка пар observer - bd_owner (оповещаемый - о ДР кого будем оповещать)
func handleSetOOCoupleForBdNotifications(ins *repository.Instance, data []byte) (string, error) {
	type bdOwner struct {
		BdOwnerId string
	}
	type bdObsOwners struct {
		BdObserverId string
		BdOwners     []bdOwner
	}

	bdOOwners := new(bdObsOwners)
	var strAnswer string

	err := json.Unmarshal(data, &bdOOwners)
	if err != nil {
		err = fmt.Errorf("handlers.handleSetOOCoupleForBdNotifications unmarshal bdObsOwners error: %v", err)
		return strAnswer, err
	}

	bdObserverId := bdOOwners.BdObserverId

	type Count struct {
		iCount int
		lock   sync.RWMutex
	}
	iCount := new(Count)
	hasErrors := false

	var wg sync.WaitGroup
	for _, bdOwner := range bdOOwners.BdOwners {
		wg.Add(1)
		iCount.lock.Lock()
		iCount.iCount++
		iCount.lock.Unlock()
		bdOwnerId := bdOwner.BdOwnerId
		go func(iCount1 *Count) {
			defer wg.Done()
			err := ins.InsertBdObsOwners(bdObserverId, bdOwnerId)
			if err != nil {
				iCount1.lock.Lock()
				iCount1.iCount--
				iCount1.lock.Unlock()
				log.Error("handlers.handleSetOOCoupleForBdNotifications запись в базу error: %v", err)
				hasErrors = true
			}
		}(iCount)
	}
	wg.Wait()
	strAnswer = "Для user-а с таб. номером " + bdObserverId + " произведено записей: " + strconv.Itoa(iCount.iCount)
	if hasErrors {
		strAnswer += ". Детализацию ошибок смотрите в логах."
	}
	return strAnswer, nil
}
