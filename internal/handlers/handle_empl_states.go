package handlers

import (
	"fmt"
	dom "mdata/internal/domain"
	"mdata/internal/repository"
	log "mdata/pkg/logging"
	"strings"
)

func handleEmployeeStateForCRUD(ins *repository.Instance, usr *dom.User, empl *dom.Employee) error {
	var err error

	selectedEmployeeStateCRUDStatus, err := checkEmployeeStateForCRUD(ins, empl)
	switch selectedEmployeeStateCRUDStatus {
	case -1:
		// error
		return err
	case 0: // не нашли, будем добавлять
		str, err := addEmployeeState(ins, empl)
		if err != nil {
			err = fmt.Errorf("не удалось добавить EmployeeState %v for user %s : with EmployeeId: %s, with EmpTabNumber: %s due to error: %v",
				empl.EmployeeCurrentState, usr.UserName, empl.EmployeeId, empl.EmpTabNumber, err)
			log.Error(err.Error())
			return err
		}
		log.Info("%v - added EmployeeState %v for user %v : with EmployeeId: %v, with EmpTabNumber: %v", str, empl.EmployeeCurrentState, usr.UserName, empl.EmployeeId, empl.EmpTabNumber)
	case 1:
		// нашли
	case 2: // need for update
		str, err := updateEmployeeState(ins, empl)
		if err != nil {
			err = fmt.Errorf("не удалось обновить EmployeeState %v for user %s : with EmployeeId: %s, with EmpTabNumber: %s due to error: %v",
				empl.EmployeeCurrentState, usr.UserName, empl.EmployeeId, empl.EmpTabNumber, err)
			log.Error(err.Error())
			return err
		}
		log.Info("%v - updated EmployeeState %v for user %v : with EmployeeId: %v, with EmpTabNumber: %v", str, empl.EmployeeCurrentState, usr.UserName, empl.EmployeeId, empl.EmpTabNumber)
	}
	return nil
}

func checkEmployeeStateForCRUD(ins *repository.Instance, empl *dom.Employee) (int, error) {
	gottenEmplState, err := ins.SelectEmployeeStateByEmplGUID(empl.EmployeeGUID)
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			return 0, err // не нашли, нужно добавлять
		}
		return -1, err // ошибка при поиске
	}

	// проверим, нужно ли обновить аттрибуты
	needUpdate := false
	if strings.TrimSpace(gottenEmplState.StateName) != strings.TrimSpace(empl.EmployeeCurrentState.StateName) {
		log.Info("NUpdt EmplState StateName for employeeGuid: %s ;  в базе: %s , загружаем: %s", empl.EmployeeGUID, gottenEmplState.StateName, empl.EmployeeCurrentState.StateName)
		needUpdate = true
	}
	if gottenEmplState.DateFrom != empl.EmployeeCurrentState.DateFrom {
		log.Info("NUpdt EmplState DateFrom for employeeGuid: %s ;  в базе: %s , загружаем: %s", empl.EmployeeGUID, gottenEmplState.DateFrom, empl.EmployeeCurrentState.DateFrom)
		needUpdate = true
	}

	if needUpdate {
		return 2, nil
	}
	return 1, nil // нашли, не нужно обновлять состояние сотрудника
}

// добавим состояние одного сотрудника:
func addEmployeeState(ins *repository.Instance, empl *dom.Employee) (string, error) {
	str, err := ins.AddEmployeeStateToDB(empl)
	if err != nil {
		return str, fmt.Errorf("handlers.addEmployeeState error: %v", err)
	}
	return str, nil
}

// обновим состояние одного сотрудника:
func updateEmployeeState(ins *repository.Instance, empl *dom.Employee) (string, error) {
	str, err := ins.UpdateEmployeeStateToDB(empl)
	if err != nil {
		return str, fmt.Errorf("handlers.updateEmployeeState error: %v", err)
	}
	return str, nil
}
