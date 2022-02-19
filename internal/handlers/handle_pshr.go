package handlers

import (
	"fmt"
	dom "mdata/internal/domain"
	"mdata/internal/repository"
	log "mdata/pkg/logging"
	"strings"
)

func handleEmployeePshrForCRUD(ins *repository.Instance, usr *dom.User, empl *dom.Employee) error {
	var err error

	selectedEmployeePshrCRUDStatus, err := checkEmployeePshrForCRUD(ins, usr.UserGUID, empl)

	switch selectedEmployeePshrCRUDStatus {
	case -1:
		// error
		return err
	case 0: // не нашли, будем добавлять
		str, err := addEmployeePshr(ins, empl)
		if err != nil {
			err = fmt.Errorf("не удалось добавить EmployeePshr %v for user %s due to error: %v", empl.EmployeePshr.PshrDescr, usr.UserName, err)
			log.Error(err.Error())
			return err
		}
		log.Info("%v - added EmployeePshr %v for user %v", str, empl.EmployeePshr.PshrDescr, usr.UserName)
	case 1:
		// нашли

	case 2: // need for update
		str, err := updateEmployeePshr(ins, empl)
		if err != nil {
			err = fmt.Errorf("не удалось обновить EmployeePshr %v for user %s due to error: %v", empl.EmployeePshr.PshrDescr, usr.UserName, err)
			log.Error(err.Error())
			return err
		}
		log.Info("%v - updated EmployeePshr %v for user %v", str, empl.EmployeePshr.PshrDescr, usr.UserName)
	}
	return nil
}

// проверим нужно ли обновить ПШР сотрудника:
func checkEmployeePshrForCRUD(ins *repository.Instance, userGuid string, emp *dom.Employee) (int, error) {
	oldEmplPshr, err := ins.GetEmplPshrByEmplGUID(emp.EmployeeGUID)
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			log.Info("NAdd Pshr for employeeGuid: %s ;  загружаем: %s", emp.EmployeeGUID, emp.EmployeePshr.PshrDescr)
			return 0, err // не нашли, нужно добавлять
		}
		return -1, err // ошибка при поиске
	}

	// проверим, нужно ли обновить аттрибуты
	needUpdate := false
	if strings.TrimSpace(oldEmplPshr.PshrGUID) != strings.TrimSpace(emp.EmployeePshr.PshrGUID) {
		log.Info("NUpdt PshrGUID for employeeGuid: %s ;  в базе: %s , загружаем: %s", emp.EmployeeGUID, oldEmplPshr.PshrGUID, emp.EmployeePshr.PshrGUID)
		needUpdate = true
	}
	if strings.TrimSpace(oldEmplPshr.PshrId) != strings.TrimSpace(emp.EmployeePshr.PshrId) {
		log.Info("NUpdt PshrId for employeeGuid: %s ;  в базе: %s , загружаем: %s", emp.EmployeeGUID, oldEmplPshr.PshrId, emp.EmployeePshr.PshrId)
		needUpdate = true
	}
	if strings.TrimSpace(oldEmplPshr.PshrDescr) != strings.TrimSpace(emp.EmployeePshr.PshrDescr) {
		log.Info("NUpdt PshrDescr for employeeGuid: %s ;  в базе: %s , загружаем: %s", emp.EmployeeGUID, oldEmplPshr.PshrDescr, emp.EmployeePshr.PshrDescr)
		needUpdate = true
	}
	if needUpdate {
		return 2, nil
	}
	return 1, nil // нашли, не нужно обновлять должность сотрудника
}

// добавим ПШР одного сотрудника:
func addEmployeePshr(ins *repository.Instance, empl *dom.Employee) (string, error) {
	str, err := ins.AddEmplPshrToDB(empl)
	if err != nil {
		return str, fmt.Errorf("handlers.AddEmplPshrToDB error: %v", err)
	}
	return str, nil
}

// удалим ПШР сотрудника:
func deleteEmployeePshr(ins *repository.Instance, oldEmplPshrGUID string) (string, error) {
	str, err := ins.DeleteEmplPshrFromDB(oldEmplPshrGUID)
	if err != nil {
		return str, fmt.Errorf("handlers.deleteEmployeePshr error: %v", err)
	}
	return str, nil

}

// обновим ПШР сотрудника:
func updateEmployeePshr(ins *repository.Instance, empl *dom.Employee) (string, error) {
	str, err := ins.UpdateEmplPshrToDB(empl)
	if err != nil {
		return str, fmt.Errorf("handlers.updateEmployeePshr error: %v", err)
	}
	return str, nil
}
