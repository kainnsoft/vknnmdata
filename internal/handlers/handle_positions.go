package handlers

import (
	"fmt"
	dom "mdata/internal/domain"
	"mdata/internal/repository"
	log "mdata/pkg/logging"
	"strings"
)

func handleEmployeePositionForCRUD(ins *repository.PostgreInstance, usr *dom.User, empl *dom.Employee) error {
	var err error

	selectedEmployeePositionCRUDStatus, err := checkEmployeePositionForCRUD(ins, usr.UserGUID, empl)

	switch selectedEmployeePositionCRUDStatus {
	case -1:
		// error
		return err
	case 0: // не нашли, будем добавлять
		str, err := addEmployeePosition(ins, empl)
		if err != nil {
			err = fmt.Errorf("не удалось добавить EmployeePosition %v for user %s due to error: %v", empl.EmployeePosition.PositionDescr, usr.UserName, err)
			log.Error(err.Error())
			return err
		}
		log.Info("%v - added EmployeePosition %v for user %v", str, empl.EmployeePosition.PositionDescr, usr.UserName)
	case 1:
		// нашли

	case 2: // need for update
		str, err := updateEmployeePosition(ins, empl)
		if err != nil {
			err = fmt.Errorf("не удалось обновить EmployeePosition %v for user %s due to error: %v", empl.EmployeePosition.PositionDescr, usr.UserName, err)
			log.Error(err.Error())
			return err
		}
		log.Info("%v - updated EmployeePosition %v for user %v", str, empl.EmployeePosition.PositionDescr, usr.UserName)
	}
	return nil
}

// проверим нужно ли обновить должность сотрудника:
func checkEmployeePositionForCRUD(ins *repository.PostgreInstance, userGuid string, emp *dom.Employee) (int, error) {
	oldEmplPos, err := ins.GetEmplPositionByEmplGUID(emp.EmployeeGUID)
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			log.Info("NAdd Position for employeeGuid: %s ;  загружаем: %s", emp.EmployeeGUID, emp.EmployeePosition.PositionDescr)
			return 0, err // не нашли, нужно добавлять
		}
		return -1, err // ошибка при поиске
	}

	// проверим, нужно ли обновить аттрибуты
	needUpdate := false
	if strings.TrimSpace(oldEmplPos.PositionGUID) != strings.TrimSpace(emp.EmployeePosition.PositionGUID) {
		log.Info("NUpdt PositionGUID for employeeGuid: %s ;  в базе: %s , загружаем: %s", emp.EmployeeGUID, oldEmplPos.PositionGUID, emp.EmployeePosition.PositionGUID)
		needUpdate = true
	}
	if strings.TrimSpace(oldEmplPos.PositionDescr) != strings.TrimSpace(emp.EmployeePosition.PositionDescr) {
		log.Info("NUpdt PositionDescr for employeeGuid: %s ;  в базе: %s , загружаем: %s", emp.EmployeeGUID, oldEmplPos.PositionDescr, emp.EmployeePosition.PositionDescr)
		needUpdate = true
	}
	if needUpdate {
		return 2, nil
	}
	return 1, nil // нашли, не нужно обновлять должность сотрудника
}

// добавим должность одного сотрудника:
func addEmployeePosition(ins *repository.PostgreInstance, empl *dom.Employee) (string, error) {
	str, err := ins.AddEmplPositionToDB(empl)
	if err != nil {
		return str, fmt.Errorf("handlers.addEmployeePosition error: %v", err)
	}
	return str, nil
}

// удалим должность сотрудника:
func deleteEmployeePosition(ins *repository.PostgreInstance, oldEmplPositionGUID string) (string, error) {
	str, err := ins.DeleteEmplPositionFromDB(oldEmplPositionGUID)
	if err != nil {
		return str, fmt.Errorf("handlers.addEmployeePosition error: %v", err)
	}
	return str, nil

}

// обновим должность сотрудника:
func updateEmployeePosition(ins *repository.PostgreInstance, empl *dom.Employee) (string, error) {
	str, err := ins.UpdateEmplPositionToDB(empl)
	if err != nil {
		return str, fmt.Errorf("handlers.updateEmployeePosition error: %v", err)
	}
	return str, nil
}
