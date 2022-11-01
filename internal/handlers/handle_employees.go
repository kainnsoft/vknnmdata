package handlers

import (
	"encoding/json"
	"fmt"
	dom "mdata/internal/domain"
	"mdata/internal/repository"
	log "mdata/pkg/logging"
	"strings"
)

//------------------------------------------------------------
// все аттрибуты
// Отдаем данные всех работающих (актуальных) сотрудников, у которых есть email-ы (все аттрибуты)
func GetAllEmailEmployeesAllAttributes(ins *repository.PostgreInstance) ([]byte, error) {
	// пока ВСЕ (не один) из базы:
	allEmployeesData, err := ins.GetAllActualEmailUsersAllAttributes()
	if err != nil {
		log.Error("GetAllEmailEmployeesAllAttributes", err)
	}

	allEmployees := dom.AGUsers{}
	allEmployees.Users = allEmployeesData

	sliceOfByte, err := json.MarshalIndent(allEmployees, "", "  ")
	if err != nil {
		log.Error("marshal GetAllEmailEmployeesAllAttributes error", err)
	}

	return sliceOfByte, nil
}

//------------------------------------------------------------
// облегчённые аттрибуты
// Отдаем данные всех работающих (актуальных) сотрудников (облегчённые аттрибуты)
func GetAllEmployeesLightVersionAttributes(ins *repository.PostgreInstance) ([]byte, error) {
	// пока ВСЕ (не один) из базы:
	allEmployeesData, err := ins.GetAllActualUsersLightVersionAttributes()
	if err != nil {
		log.Error("GetAllEmployeesLightVersionAttributes", err)
	}

	allEmployees := dom.AGUsers{}
	allEmployees.Users = allEmployeesData

	sliceOfByte, err := json.MarshalIndent(allEmployees, "", "  ")
	if err != nil {
		log.Error("marshal GetAllEmployeesLightVersionAttributes error", err)
	}

	return sliceOfByte, nil
}

// Отдаем данные всех работающих (актуальных) сотрудников, у которых есть email-ы (все аттрибуты)
func GetAllEmailEmployeesLightVersionAttributes(ins *repository.PostgreInstance) ([]byte, error) {
	// пока ВСЕ (не один) из базы:
	allEmployeesData, err := ins.GetAllActualEmailUsersLightVersionAttributes()
	if err != nil {
		log.Error("GetAllEmailEmployeesLightVersionAttributes", err)
	}

	allEmployees := dom.AGUsers{}
	allEmployees.Users = allEmployeesData

	sliceOfByte, err := json.MarshalIndent(allEmployees, "", "  ")
	if err != nil {
		log.Error("marshal GetAllEmailEmployeesLightVersionAttributes error", err)
	}

	return sliceOfByte, nil
}

//------------------------------------------------------------
// переберем всех сотрудников пользователя проверим каждого на предмет: нужно ли его обновить (или может, добавить...) и произведем нужное действие:
func handleAllUserEmployeesForCRUD(ins *repository.PostgreInstance, usr *dom.User) error {
	for _, emp := range usr.Employees {
		err := handleSingleEmployeeForCRUD(ins, usr, &emp)
		if err != nil {
			// в log уже записали, поэтому продолжим обрабатывать других user-ов
			continue
		}
	}
	return nil
}

func handleSingleEmployeeForCRUD(ins *repository.PostgreInstance, usr *dom.User, empl *dom.Employee) error {
	var err error

	selectedEmployeeCRUDStatus, err := checkSingleEmployeeForCRUD(ins, usr.UserGUID, empl)
	switch selectedEmployeeCRUDStatus {
	case -1:
		// error
		return err
	case 0: // не нашли, будем добавлять
		err = addEmployee(ins, usr, empl) // TODO
		if err != nil {
			return err
		}

	case 1:
		// нашли

	case 2: // need for update
		err = updateEmployee(ins, usr, empl) // TODO
		if err != nil {
			return err
		}
		log.Info("updated employee for user %v : %v, %v, %v, %v, %v", usr.UserName, empl.EmployeeGUID, empl.EmployeeId, empl.EmpTabNumber, empl.EmployeeAdress, empl.Employment)
	}
	// check for update departament: подразделение добавлять не будем. Таблица подразделений заполнянтся не зависимо от сотрудников. У сотрудника есть ссылка (GUID) на подразделение.
	// check for update статус сотрудника:
	err = handleEmployeeStateForCRUD(ins, usr, empl)
	if err != nil {
		return err
	}
	// добавим должность сотруднику
	err = handleEmployeePositionForCRUD(ins, usr, empl)
	if err != nil {
		return err
	}
	// добавим ПШР сотруднику
	err = handleEmployeePshrForCRUD(ins, usr, empl)
	if err != nil {
		return err
	}

	return nil
}

// TODO
// проверим нужно ли обновить данные одного конкретного сотрудника:
func checkSingleEmployeeForCRUD(ins *repository.PostgreInstance, userGuid string, emp *dom.Employee) (int, error) {
	oldEmployee, err := ins.SelectSingleEmployeeWithAttribsByGUID(emp.EmployeeGUID)
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			return 0, err // не нашли, нужно добавлять
		}
		return -1, err // ошибка при поиске
	}

	// проверим, нужно ли обновить аттрибуты
	needUpdate := false
	if strings.TrimSpace(oldEmployee.EmployeeId) != strings.TrimSpace(emp.EmployeeId) {
		log.Info("NUpdt EmployeeId for employeeGuid: %s ;  в базе: %s , загружаем: %s", emp.EmployeeGUID, oldEmployee.EmployeeId, emp.EmployeeId)
		needUpdate = true
	}
	if strings.TrimSpace(oldEmployee.EmpTabNumber) != strings.TrimSpace(emp.EmpTabNumber) {
		log.Info("NUpdt EmpTabNumber for employeeGuid: %s ;  в базе: %s , загружаем: %s", emp.EmployeeGUID, oldEmployee.EmpTabNumber, emp.EmpTabNumber)
		needUpdate = true
	}
	if strings.TrimSpace(oldEmployee.EmployeeAdress) != strings.TrimSpace(emp.EmployeeAdress) {
		log.Info("NUpdt EmployeeAdress for employeeGuid: %s ;  в базе: %s , загружаем: %s", emp.EmployeeGUID, oldEmployee.EmployeeAdress, emp.EmployeeAdress)
		needUpdate = true
	}
	if strings.TrimSpace(oldEmployee.Employment) != strings.TrimSpace(emp.Employment) {
		log.Info("NUpdt Employment for employeeGuid: %s ;  в базе: %s , загружаем: %s", emp.EmployeeGUID, oldEmployee.Employment, emp.Employment)
		needUpdate = true
	}
	if strings.TrimSpace(oldEmployee.EmployeeDepartament.DepartamentGUID) != strings.TrimSpace(emp.EmployeeDepartament.DepartamentGUID) {
		log.Info("NUpdt empl.Departament for employeeGuid: %s ;  в базе: %s , загружаем: %s", emp.EmployeeGUID, oldEmployee.EmployeeDepartament.DepartamentGUID, emp.EmployeeDepartament.DepartamentGUID)
		needUpdate = true
	}
	if needUpdate {
		return 2, nil
	}
	return 1, nil // нашли, не нужно обновлять сотрудника, но нужно проверить его подразделение, должность и состояние
}

// добавим одного сотрудника пользователя с подразделением, долж. и т.д.:
func addEmployee(ins *repository.PostgreInstance, u *dom.User, empl *dom.Employee) error {
	// сам сотрудник
	str, err := ins.AddEmployeeToDB(u.UserGUID, empl)
	if err != nil {
		log.Error("Не удалось создать сотрудника для пользователя %s с кодом %s", u.UserName, u.UserID)
		return fmt.Errorf("handlers.addEmployee error: %v", err)
	}
	log.Info("%s - создан сотрудник для пользователя %s с кодом %s", str, u.UserName, u.UserID)

	return nil
}

// обновим одного сотрудника пользователя; только его - без подразделения, долж. и т.д.:
func updateEmployee(ins *repository.PostgreInstance, usr *dom.User, empl *dom.Employee) error {
	// сам сотрудник
	str, err := ins.UpdateEmployeeInDB(usr.UserGUID, empl)
	if err != nil {
		log.Error("Не удалось обновить сотрудника для пользователя %s с кодом %s", usr.UserName, usr.UserID)
		return fmt.Errorf("handlers.updateEmployee error: %v", err)
	}
	log.Info("%s - обновлён сотрудник для пользователя %s с кодом %s", str, usr.UserName, usr.UserID)

	return nil
}
