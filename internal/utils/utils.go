package utils

import (
	"fmt"
	"mdata/internal/domain"
	"mdata/internal/repository"
	"os"
	"reflect"
	"strconv"
	"time"

	log "mdata/internal/logging"
)

// Отправка log-а в бухгалтерию:
func SendEmailToBuch(ins *repository.Instance) error {
	t := time.Now()
	if t.Weekday() == time.Friday {
		// адресаты
		recipients, err := ins.GetUserEmailsByNotificationsTypes(2) // notification_type = 'В бухгалтерию о загрузке email-ов в 1С:ЗУП', id = 2
		if err != nil {
			return err
		}

		admins, err := ins.GetUserEmailsByNotificationsTypes(1)
		if err != nil {
			log.Error("notifications handlers.SendEmailToBuch GetBccAdmin error: %v", err)
		}
		bccAdmin := admins[0]
		if bccAdmin == "" {
			log.Error("notifications handlers.SendEmailToBuch GetBccAdmin error: no bccAdmin", err)
		}

		// тема
		subject := "Emails file"
		// тело
		t := time.Now()
		_, thisWeek := t.ISOWeek()
		weekNumber := strconv.Itoa(thisWeek)
		formatted := fmt.Sprintf("%02d.%02d.%d", t.Day(), t.Month(), t.Year())
		body := fmt.Sprintf("Email-ы сотрудников, заполненные на %s неделе. Отправлено - %vг.", weekNumber, formatted)
		// отправка
		err = repository.SendMailToRecipient(recipients, bccAdmin, subject, body, log.FBuchName)
		if err != nil {
			return err
		}
		if err = os.Truncate(log.FBuchName, 0); err != nil { // очистить содержимое файла
			return err
		}
	} else {
		return fmt.Errorf("today is not friday")
	}
	return nil
}

// Отправка сообщений админам 1С о необходимости промониторить работу по автоматическому созданию пользователей:
func SendEmailTo1CAdmins(ins *repository.Instance, usersToExchangeSlice []domain.User, strInErr string) {
	// адресаты
	recipients, err := ins.GetUserEmailsByNotificationsTypes(3) // notification_type = 'В 1C:CreateUser о новом user-е с email', id = 3
	if err != nil {
		log.Error("notifications handlers.SendEmailTo1CAdmins GetRecipients error: %v", err)
	}
	// скрытая копия
	admins, err := ins.GetUserEmailsByNotificationsTypes(1)
	if err != nil {
		log.Error("notifications handlers.SendEmailTo1CAdmins GetBccAdmin error: %v", err)
	}
	bccAdmin := admins[0]
	if bccAdmin == "" {
		log.Error("notifications handlers.SendEmailTo1CAdmins GetBccAdmin error: no bccAdmin: %v", err)
	}

	// тема
	subject := "В 1C:CreateUser о новом user-е с email"
	// тело
	body := "Попытка выгрузить следующих пользователей для автоматического создания \n"
	for _, user := range usersToExchangeSlice {
		body = body + "-------------------------- \n"
		body = body + "        " + user.UserName + " (код = " + user.UserID + ")" + " \r\n"
	}
	body = body + "-------------------------- \n"
	if strInErr == "" {
		body = body + "Завершилась успехом \n"
	} else {
		body = body + "Завершилась с ошибкой: \n"
		body = body + strInErr + "\n"
	}
	// отправка
	err = repository.SendMailToRecipient(recipients, bccAdmin, subject, body, "")
	if err != nil {
		log.Error("notifications handlers.SendEmailTo1CAdmins SendMailToRecipient error: %v", err)
	}

	log.Info("notifications handlers.SendEmailTo1CAdmins OK")
}

//
//https://medium.com/wesionary-team/reflections-tutorial-query-string-to-struct-parser-in-go-b2f858f99ea1
func shouldBeStruct(dyn interface{}) {
	typeDyn := reflect.TypeOf(dyn)
	fmt.Printf("Input kind is: %v \n", typeDyn.Kind())
	if typeDyn.Kind() != reflect.Struct {
		fmt.Printf("Struct cannot be found: input is:  %v \n", typeDyn.Kind())
	}
}

// ******************************************************************************************
// работа с датой

// начало дня
func StartOfThisDay(now time.Time) time.Time {
	currentYear, currentMonth, currentDay := now.Date()
	currentLocation := now.Location()
	startThisDay := time.Date(currentYear, currentMonth, currentDay, 0, 0, 0, 0, currentLocation)
	return startThisDay
}

func NextWeek(now time.Time) (nextWeekFrom time.Time, nextWeekTo time.Time) {
	startThisDay := StartOfThisDay(now)
	nextWeekFrom = startThisDay.AddDate(0, 0, int(7-startThisDay.Weekday()+1))
	nextWeekTo = nextWeekFrom.AddDate(0, 0, 7).Add(-1 * time.Second)
	return
}

func NextMonth(now time.Time) (nextMonthFrom time.Time, nextMonthTo time.Time) {
	currentYear, currentMonth, _ := now.Date()
	currentLocation := now.Location()
	nextMonthFrom = time.Date(currentYear, currentMonth, 1, 0, 0, 0, 0, currentLocation).AddDate(0, 1, 0)
	nextMonthTo = nextMonthFrom.AddDate(0, 1, 0).Add(-1 * time.Second)
	return
}

func GetNextWeekTemplStruct(now time.Time) domain.NextWeekStruct {
	nextWeekFrom, _ := NextWeek(now) // след. неделя начинается с (даты)
	nextWeek := domain.NextWeekStruct{}
	layout := "2006-01-02"
	for i := 0; i < 7; i++ {
		t := nextWeekFrom.AddDate(0, 0, i)
		curTempl := t.Format(layout)[4:10]
		switch t.Weekday() {
		case time.Monday:
			nextWeek.Monday = curTempl
		case time.Tuesday:
			nextWeek.Tuesday = curTempl
		case time.Wednesday:
			nextWeek.Wednesday = curTempl
		case time.Thursday:
			nextWeek.Thursday = curTempl
		case time.Friday:
			nextWeek.Friday = curTempl
		case time.Saturday:
			nextWeek.Saturday = curTempl
		case time.Sunday:
			nextWeek.Sunday = curTempl
		}
	}
	return nextWeek
}
