package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	dom "mdata/internal/domain"
	"mdata/internal/repository"
	log "mdata/pkg/logging"
)

//*********************************************************************************
// подразделения

// ping-метод. Посмотреть подразделения из 1С:ЗУП (on-line)
func HandleZupPingAllDepartaments(w http.ResponseWriter, body []byte) {
	fmt.Fprint(w, string(body))
}

// Функция для записи подразделений в базу из 1С:ЗУП
func handleAllDepartamentForCRUD(ins *repository.Instance, data []byte) (string, error) {
	var gettingDeps dom.Departaments
	var strAnswer string

	err := json.Unmarshal(data, &gettingDeps)
	if err != nil {
		err = fmt.Errorf("handlers.handleAllDepartamentForCRUD unmarshal gettingDeps error: %v", err)
		return strAnswer, err
	}

	var depsCount int
	for i, dep := range gettingDeps.Departaments {
		depsCount = i
		if i%300 == 0 {
			log.Info("i = %d", i)
		}
		_, err := handleSingleDepartamentForCRUD(ins, dep)
		if err != nil {
			continue
		}
	}

	strAnswer = "AllDepartaments - " + strconv.Itoa(depsCount)
	return strAnswer, nil // TODO: error := "some(count) errors occure"
}

// Функция для записи подразделения в базу, если не нашли такого по GUID, либо обновления, если нашли
func handleSingleDepartamentForCRUD(ins *repository.Instance, dep dom.Departament) (string, error) {
	var str string
	var err error

	// проверим, есть ли подр-е в БД
	statusSelectedDep, err := checkSingleDepartamentForCRUD(ins, &dep)
	switch statusSelectedDep {
	case -1:
		// error
		err = fmt.Errorf("неопознанная ошибка при обновлении подразделения %s с кодом %s : %v", dep.DepartamentDescr, dep.DepartamentIdZUP, err) // TODO: здесь ли нужно логирование?
		return str, err                                                                                                                          // errors.New("handlers.checkSingleDepartamentForCRUD unknown error occures")
	case 0: // не нашли, будем добавлять
		str, err = ins.AddDepartament(&dep)
		if err != nil {
			err = fmt.Errorf("не удалось создать подразделение %s с кодом %s : %v", dep.DepartamentDescr, dep.DepartamentIdZUP, err) // TODO: здесь ли нужно логирование?
			return str, err                                                                                                          // fmt.Errorf("handlers.checkSingleDepartamentForCRUD error: %v", err)
		}
		str = fmt.Sprintf("%s - создано подразделение %s с кодом %s", str, dep.DepartamentDescr, dep.DepartamentIdZUP) // TODO: здесь ли нужно логирование?
		log.Info(str)
	case 1:
		// нашли (ничего не делаем)
	case 2: // need for update
		str, err = ins.UpdateDepartament(&dep)
		if err != nil {
			err = fmt.Errorf("не удалось обновить подразделение %s с кодом %s : %v", dep.DepartamentDescr, dep.DepartamentIdZUP, err) // TODO: здесь ли нужно логирование?
			return str, err                                                                                                           // fmt.Errorf("handlers.checkSingleDepartamentForCRUD error: %v", err)
		}
		str = fmt.Sprintf("%s - обновлено подразделение %s с кодом %s", str, dep.DepartamentDescr, dep.DepartamentIdZUP) // TODO: здесь ли нужно логирование?
		log.Info(str)
	}

	return str, nil
}

func checkSingleDepartamentForCRUD(ins *repository.Instance, dep *dom.Departament) (int, error) {
	gottenDep, err := ins.SelectDepByGUID(dep.DepartamentGUID)
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			return 0, err // не нашли, нужно добавлять
		}
		return -1, err // ошибка при поиске
	}

	// проверим, нужно ли обновить аттрибуты
	needUpdate := false
	if strings.TrimSpace(gottenDep.DepartamentDescr) != strings.TrimSpace(dep.DepartamentDescr) {
		log.Info("NUpdt DepDescr for Dep: %v  with ZUP id: %v в базе MD : %v,  надо: %v", dep.DepartamentDescr, dep.DepartamentIdZUP, gottenDep.DepartamentDescr, dep.DepartamentDescr)
		needUpdate = true
	}
	if strings.TrimSpace(gottenDep.DepartamentIdZUP) != strings.TrimSpace(dep.DepartamentIdZUP) {
		log.Info("NUpdt DepIdZUP for Dep: %v  with ZUP id: %v в базе MD : %v,  надо: %v", dep.DepartamentDescr, dep.DepartamentIdZUP, gottenDep.DepartamentIdZUP, dep.DepartamentIdZUP)
		needUpdate = true
	}
	if strings.TrimSpace(gottenDep.DepartamentParentIdZUP) != strings.TrimSpace(dep.DepartamentParentIdZUP) {
		log.Info("NUpdt DepParentIdZUP for Dep: %v  with ZUP id: %v в базе MD : %v,  надо: %v", dep.DepartamentDescr, dep.DepartamentIdZUP, gottenDep.DepartamentParentIdZUP, dep.DepartamentParentIdZUP)
		needUpdate = true
	}
	if gottenDep.DepartamentNotUsedFrom != dep.DepartamentNotUsedFrom {
		log.Info("NUpdt DepNotUsedFrom for Dep: %v  with ZUP id: %v в базе MD : %v,  надо: %v", dep.DepartamentDescr, dep.DepartamentIdZUP, gottenDep.DepartamentNotUsedFrom, dep.DepartamentNotUsedFrom)
		needUpdate = true
	}
	if needUpdate {
		return 2, nil
	}

	return 1, nil // просто нашли, обновлять не нужно, добавлять тем более )
}
