package domain

import (
	"database/sql/driver"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"
)

//--------------------------------------------
// Кастомный тип для даты UserBirthday.
type CastDate struct {
	time.Time
}

func (c *CastDate) DateToString() string {
	layout := "2006-01-02"
	str := c.Time.Format(layout)
	return str
}

func (c *CastDate) UnmarshalJSON(b []byte) (err error) {
	//layout := "2006-01-02T15:04:05"
	layout := "2006-01-02"
	s := strings.Trim(string(b), "\"") // remove quotes
	if s == "null" {
		return
	}
	s1 := s[0:10]
	c.Time, err = time.Parse(layout, s1)
	if err != nil {
		return err
	}
	return
}

func (t *CastDate) MarshalJSON() ([]byte, error) {
	str := fmt.Sprintf("\"%s\"", time.Time(t.Time).Format("2006-01-02"))
	return []byte(str), nil
}

func (t *CastDate) Scan(src interface{}) error {
	switch v := src.(type) {
	case string:
		var err error
		t.Time, err = pq.ParseTimestamp(time.UTC, v)
		if err != nil {
			return err
		}
	case []byte:
		var err error
		t.Time, err = pq.ParseTimestamp(time.UTC, string(v))
		if err != nil {
			return err
		}
	case time.Time:
		t.Time = v
		// var err error
		// t.Time, err = time.Parse("2006-01-02", v.Format("2006-01-02"))
		// if err != nil {
		// 	return err
		// }

	default:
		return fmt.Errorf("can't read %T into myTime", v)
	}
	return nil
}

func (t *CastDate) Value() (driver.Value, error) {
	return driver.Value(t.Time), nil
}

//--------------------------------------------
// все атрибуты
type AGUsers struct {
	Users []User `json:"users"`
}
type User struct {
	UserGUID     string     `json:"userGuid"`
	UserName     string     `json:"userName"`
	UserID       string     `json:"userId"`
	UserBirthday CastDate   `json:"userBirthday"`
	UserEmail    string     `json:"userEmail"`
	Employees    []Employee `json:"employees"`
}
type Employee struct {
	EmployeeGUID         string           `json:"employeeGuid"`
	EmployeeId           string           `json:"employeeId"`
	Employment           string           `json:"employment"` // тип работы ("Основное место работы", "Внутреннее совместительство" и т.д.)
	EmpTabNumber         string           `json:"tabNumber"`
	EmployeePosition     Position         `json:"position"`
	EmployeePshr         Pshr             `json:"positionShr"`
	EmployeeDepartament  Departament      `json:"departament"`
	EmployeeCurrentState EmplCurrentState `json:"currentState"`
	EmployeeAdress       string           `json:"employeeAdress"`
}
type Position struct {
	PositionGUID  string `json:"positionGuid"`
	PositionDescr string `json:"positionDescr"`
}

type Pshr struct {
	PshrGUID  string `json:"pshrGuid"`
	PshrId    string `json:"pshrId"`
	PshrDescr string `json:"pshrDescr"`
}

type EmplCurrentState struct {
	StateName string   `json:"stateName"`
	DateFrom  CastDate `json:"dateFrom"`
}
