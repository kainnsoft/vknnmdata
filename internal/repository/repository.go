package repository

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"mdata/internal/domain"
	log "mdata/pkg/logging"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

//***************************************************************************************
// Для работы с БД
const queryTimeout = 100

//Функция для более удобного создания строки подключения к БД
func NewPoolConfig(cfg *domain.Config) (*pgxpool.Config, error) {
	connStr := fmt.Sprintf("%s://%s:%s@%s:%s/%s?sslmode=disable&connect_timeout=%d",
		"postgres",
		url.QueryEscape(cfg.DBUsername),
		url.QueryEscape(cfg.DBPassword),
		cfg.DBHost,
		cfg.DBPort,
		cfg.DataBaseName,
		cfg.DBTimeout)

	poolConfig, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, err
	}

	return poolConfig, nil
}

//Функция-обертка для создания подключения с помощью пула
func NewConnection(poolConfig *pgxpool.Config) (*pgxpool.Pool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	conn, err := pgxpool.ConnectConfig(ctx, poolConfig)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

// основная структура Instance, которая принимает в зависимость подключение.
// Благодаря этому, мы имеем возможность делать запросы в базу прямо из нашего пакета repository
type Instance struct {
	Db *pgxpool.Pool
}

func DBPing(i *Instance) (string, error) { // pool *pgxpool.Pool
	// Func "Exec" performs query to DB
	ctx, cancel := context.WithTimeout(context.Background(), queryTimeout*time.Millisecond)
	defer cancel()
	_, err := i.Db.Exec(ctx, ";")
	if err != nil {
		return "", fmt.Errorf("DB ping error: %v", err)
	}
	return "Connection OK! Query OK!", nil
}

//***************************************************************************************

// Пользователи  (физ.лица)

// get user by GUID
func (i *Instance) SelectUserByGUID(userGUID string) (domain.User, error) {

	var usr domain.User = domain.User{}

	rows, err := i.Db.Query(context.Background(), "select user_guid, user_name, user_id, user_birthday, email from users where user_guid=$1;",
		userGUID)
	if err == pgx.ErrNoRows {
		err = fmt.Errorf("no rows: repository.SelectUserByGUID error: %v", err) // "no rows" - не удалять - по ним дальше определяем добавление
		log.Error(err.Error())
		return usr, err
	} else if err != nil {
		log.Error("repository.SelectUserByGUID error: %v", err)
		return usr, err
	}
	defer rows.Close()

	// такое тоже бывает, не определилось на стадии pgx.ErrNoRows
	if !rows.Next() {
		err = errors.New("repository.SelectUserByGUID error: no rows") // "no rows" - не удалять - по ним дальше определяем добавление
		log.Error(err.Error())
		return usr, err
	} else {
		curUser := new(domain.User)
		rows.Scan(&curUser.UserGUID,
			&curUser.UserName,
			&curUser.UserID,
			&curUser.UserBirthday,
			&curUser.UserEmail)
		usr = *curUser
	}

	return usr, nil
}

func (i *Instance) AddUserToDB(v *domain.User, currEmail string) (string, error) {
	curBirthday := v.UserBirthday.DateToString()
	commandTag, err := i.Db.Exec(context.Background(), "INSERT INTO users (user_guid, user_name, user_id, user_birthday, email) VALUES ($1, $2, $3, $4, $5);",
		v.UserGUID,
		v.UserName,
		strings.TrimSpace(v.UserID),
		curBirthday,
		currEmail)

	if err != nil {
		log.Error("repository.AddUser error %v", err)
		return commandTag.String(), err
	}

	return commandTag.String(), nil
}

func (i *Instance) UpdateUserInDB(v *domain.User, currEmail string) (string, error) {
	curBirthday := v.UserBirthday.DateToString()
	commandTag, err := i.Db.Exec(context.Background(),
		"UPDATE users set user_name=$1, user_id=$2, user_birthday=$3, email=$4 where user_guid=$5;",
		v.UserName,
		strings.TrimSpace(v.UserID),
		curBirthday, //v.UserBirthday,
		currEmail,
		v.UserGUID)

	if err != nil {
		log.Error("repository.UpdateUser error %v", err)
		return commandTag.String(), err
	}

	return commandTag.String(), nil
}

func (i *Instance) GetAllUsersWithEmail() ([]domain.User, error) {

	usersSlice := make([]domain.User, 0)

	rows, err := i.Db.Query(context.Background(), "select user_guid, user_name, user_id, email from users where email <> '';")
	if err == pgx.ErrNoRows {
		fmt.Println("No rows")
		return usersSlice, err
	} else if err != nil {
		fmt.Println(err)
		return usersSlice, err
	}
	defer rows.Close()

	for rows.Next() {
		curUser := new(domain.User)
		rows.Scan(&curUser.UserGUID, &curUser.UserName, &curUser.UserID, &curUser.UserEmail)
		usersSlice = append(usersSlice, *curUser)
	}

	return usersSlice, nil
}

// вернём всех пользователей (физ.лиц) и уволенные тоже:
func (i *Instance) GetAllUsersFromDB() ([]domain.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), queryTimeout*time.Millisecond)
	defer cancel()

	const usrs_query = "select user_guid, user_name, user_id, user_birthday, email from users;"

	allUsersSlice := make([]domain.User, 0, 5000)

	rows, err := i.Db.Query(ctx, usrs_query)
	if err == pgx.ErrNoRows {
		fmt.Println("No rows")
		return allUsersSlice, err
	} else if err != nil {
		fmt.Println(err)
		return allUsersSlice, err
	}
	defer rows.Close()

	for rows.Next() {
		curUser := new(domain.User)
		rows.Scan(&curUser.UserGUID, &curUser.UserName, &curUser.UserID, &curUser.UserBirthday, &curUser.UserEmail)

		usersEmplSlice, _ := selectEmployeesWithAttribsByUserGUID(i.Db, curUser.UserGUID)
		curUser.Employees = usersEmplSlice
		allUsersSlice = append(allUsersSlice, *curUser)
	}
	return allUsersSlice, nil
}

// вернём массив пользователей с подходящими кодами (табельными номерами):
func (i *Instance) GetUsersByTabNo(tabno string) ([]domain.User, error) {

	const usrs_query = "select user_guid, user_name, user_id, user_birthday, email from users where user_id like $1;"

	usersByTabNoSlice := make([]domain.User, 0)

	rows, err := i.Db.Query(context.Background(), usrs_query+"", "%"+tabno+"%") //  '%'||$1||'%';", tabno) //
	if err == pgx.ErrNoRows {
		fmt.Println("No rows")
		return usersByTabNoSlice, err
	} else if err != nil {
		fmt.Println(err)
		return usersByTabNoSlice, err
	}
	defer rows.Close()

	for rows.Next() {
		curUser := new(domain.User)
		rows.Scan(&curUser.UserGUID, &curUser.UserName, &curUser.UserID, &curUser.UserBirthday, &curUser.UserEmail)

		usersEmplSlice, _ := selectEmployeesWithAttribsByUserGUID(i.Db, curUser.UserGUID)
		curUser.Employees = usersEmplSlice
		usersByTabNoSlice = append(usersByTabNoSlice, *curUser)
	}
	return usersByTabNoSlice, nil
}

// вернём массив всех(в т.ч. уволенных) пользователей с подходящими ФИО:
func (i *Instance) GetUsersByUserName(userName string) ([]domain.User, error) {

	const usrs_query = "select user_guid, user_name, user_id, user_birthday, email from users where user_name ilike $1;"

	usersByUserNameSlice := make([]domain.User, 0)

	rows, err := i.Db.Query(context.Background(), usrs_query, "%"+userName+"%") //  '%'||$1||'%';", userName) //
	if err == pgx.ErrNoRows {
		fmt.Println("No rows")
		return usersByUserNameSlice, err
	} else if err != nil {
		fmt.Println(err)
		return usersByUserNameSlice, err
	}
	defer rows.Close()

	for rows.Next() {
		curUser := new(domain.User)
		rows.Scan(&curUser.UserGUID, &curUser.UserName, &curUser.UserID, &curUser.UserBirthday, &curUser.UserEmail)

		usersEmplSlice, _ := selectEmployeesWithAttribsByUserGUID(i.Db, curUser.UserGUID)
		curUser.Employees = usersEmplSlice
		usersByUserNameSlice = append(usersByUserNameSlice, *curUser)
	}
	return usersByUserNameSlice, nil
}

// get users by slice of GUID
func (i *Instance) GetUsersBySliceOfGUID(userGUIDSlice []string) ([]domain.User, error) {

	usr := make([]domain.User, 0, len(userGUIDSlice))

	queryStr := "select user_guid, user_name, user_id, user_birthday, email from users where user_guid = ANY($1::uuid[])"
	param := "{" + strings.Join(userGUIDSlice, ",") + "}"

	rows, err := i.Db.Query(context.Background(), queryStr, param)
	if err == pgx.ErrNoRows {
		err = fmt.Errorf("no rows: repository.SelectUserByGUID error: %v", err) // "no rows" - не удалять - по ним дальше определяем добавление
		log.Error(err.Error())
		return usr, err
	} else if err != nil {
		log.Error("repository.SelectUserByGUID error: %v", err)
		return usr, err
	}
	defer rows.Close()

	for rows.Next() {
		curUser := new(domain.User)
		rows.Scan(&curUser.UserGUID, &curUser.UserName, &curUser.UserID, &curUser.UserBirthday, &curUser.UserEmail)
		usr = append(usr, *curUser)
	}

	return usr, nil
}

// вернём список пользователей (физ.лиц) по заданному списку guid-ов все атрибуты:
func (i *Instance) GetCastomUserListAllAttributes(userGUIDSlice []string) ([]domain.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), queryTimeout*time.Millisecond)
	defer cancel()

	const usrs_query = commonQueryAllAttributes + " where user_guid = ANY($1::uuid[])"
	param := "{" + strings.Join(userGUIDSlice, ",") + "}"

	allUsersSlice := make([]domain.User, 0)

	rows, err := i.Db.Query(ctx, usrs_query, param)
	if err == pgx.ErrNoRows {
		fmt.Println("No rows")
		return allUsersSlice, err
	} else if err != nil {
		fmt.Println(err)
		return allUsersSlice, err
	}
	defer rows.Close()

	allUsersSlice = handlRowsAllAttributes(rows)

	return allUsersSlice, nil
}

//***************************************************************************************
//---------------------------------------
// все аттрибуты
const commonQueryAllAttributes = "select usr.user_guid, usr.user_name, usr.user_id, usr.user_birthday, usr.email, " +
	" empl.employee_guid, empl.employee_id, empl.employee_tabno, empl.employment, empl.employee_adress, " +
	" dep.departament_guid, dep.zup_id, dep.departament_descr, dep.zup_parent_id, dep.departament_parent_guid, dep.zup_not_used_from, " +
	" pos.position_guid, pos.position_descr, " +
	" pshr.pshr_guid, pshr.pshr_id, pshr.pshr_descr, " +
	" emplCS.state_descr, emplCS.state_date_from " +
	"    from users usr " +
	"        left join employees empl on usr.user_guid=empl.employee_user " +
	"        left join departaments dep on empl.employee_departament=dep.departament_guid " +
	"        left join positions pos on pos.employee_guid=empl.employee_guid" +
	"        left join pshr_list pshr on pshr.employee_guid=empl.employee_guid" +
	"        left join employee_states emplCS on emplCS.employee_guid=empl.employee_guid "

func handlRowsAllAttributes(rows pgx.Rows) []domain.User {
	allUsersSlice := make([]domain.User, 0, 1000)
	curUser := new(domain.User)
	var oldUser = domain.User{}
	empSlice := make([]domain.Employee, 0, 2)

	for rows.Next() {

		curEmpl := new(domain.Employee)
		curDep := new(domain.Departament)
		curPos := new(domain.Position)
		curPshr := new(domain.Pshr)
		curEmplSt := new(domain.EmplCurrentState)
		rows.Scan(&curUser.UserGUID, &curUser.UserName, &curUser.UserID, &curUser.UserBirthday, &curUser.UserEmail,
			&curEmpl.EmployeeGUID, &curEmpl.EmployeeId, &curEmpl.EmpTabNumber, &curEmpl.Employment, &curEmpl.EmployeeAdress,
			&curDep.DepartamentGUID, &curDep.DepartamentIdZUP, &curDep.DepartamentDescr, &curDep.DepartamentParentIdZUP, &curDep.DepartamentParentGUID, &curDep.DepartamentNotUsedFrom,
			&curPos.PositionGUID, &curPos.PositionDescr,
			&curPshr.PshrGUID, &curPshr.PshrId, &curPshr.PshrDescr,
			&curEmplSt.StateName, &curEmplSt.DateFrom)

		curEmpl.EmployeeDepartament = *curDep
		curEmpl.EmployeePosition = *curPos
		curEmpl.EmployeePshr = *curPshr
		curEmpl.EmployeeCurrentState = *curEmplSt

		if (oldUser.UserGUID != "") && (curUser.UserGUID != oldUser.UserGUID) {
			oldUser.Employees = empSlice
			allUsersSlice = append(allUsersSlice, oldUser)
			empSlice = make([]domain.Employee, 0, 2)
		}

		empSlice = append(empSlice, *curEmpl)
		oldUser = *curUser
	}
	oldUser.Employees = empSlice
	allUsersSlice = append(allUsersSlice, oldUser)

	return allUsersSlice
}

// вернём всех работающих пользователей (физ.лиц) все атрибуты:
func (i *Instance) GetAllActualUsersAllAttributes() ([]domain.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), queryTimeout*time.Millisecond)
	defer cancel()

	const usrs_query = commonQueryAllAttributes + " where state_descr ilike $1;"

	allUsersSlice := make([]domain.User, 0)

	rows, err := i.Db.Query(ctx, usrs_query, "%Работ%")
	if err == pgx.ErrNoRows {
		fmt.Println("No rows")
		return allUsersSlice, err
	} else if err != nil {
		fmt.Println(err)
		return allUsersSlice, err
	}
	defer rows.Close()

	allUsersSlice = handlRowsAllAttributes(rows)

	return allUsersSlice, nil
}

// вернём всех работающих пользователей (физ.лиц), у которых есть email-ы (все аттрибуты):
func (i *Instance) GetAllActualEmailUsersAllAttributes() ([]domain.User, error) {

	const usrs_query = commonQueryAllAttributes + " where state_descr ilike $1 and email<>'';"

	allUsersSlice := make([]domain.User, 0)

	rows, err := i.Db.Query(context.Background(), usrs_query, "%Работ%")
	if err == pgx.ErrNoRows {
		fmt.Println("No rows")
		return allUsersSlice, err
	} else if err != nil {
		fmt.Println(err)
		return allUsersSlice, err
	}
	defer rows.Close()

	allUsersSlice = handlRowsAllAttributes(rows)

	return allUsersSlice, nil
}

// вернём массив только работающих пользователей с подходящими кодами - табельными номерами (?tabno=8337) все атрибуты:
func (i *Instance) GetActualUsersByTabNoAllAttributes(tabno string) ([]domain.User, error) {

	const usrs_query = commonQueryAllAttributes + " where user_id ilike $1 and state_descr ilike $2;"

	usersByTabNoSlice := make([]domain.User, 0)

	rows, err := i.Db.Query(context.Background(), usrs_query, "%"+tabno+"%", "%Работ%")
	if err == pgx.ErrNoRows {
		fmt.Println("No rows")
		return usersByTabNoSlice, err
	} else if err != nil {
		fmt.Println(err)
		return usersByTabNoSlice, err
	}
	defer rows.Close()

	usersByTabNoSlice = handlRowsAllAttributes(rows)

	return usersByTabNoSlice, nil
}

// вернём массив только работающих пользователей с подходящими ФИО (?name=захаро) все атрибуты:
func (i *Instance) GetActualUsersByUserNameAllAttributes(userName string) ([]domain.User, error) {

	const usrs_query = commonQueryAllAttributes + " where user_name ilike $1 and state_descr ilike $2;"

	usersByUserNameSlice := make([]domain.User, 0)

	rows, err := i.Db.Query(context.Background(), usrs_query, "%"+userName+"%", "%Работ%")
	if err == pgx.ErrNoRows {
		fmt.Println("No rows")
		return usersByUserNameSlice, err
	} else if err != nil {
		fmt.Println(err)
		return usersByUserNameSlice, err
	}
	defer rows.Close()

	usersByUserNameSlice = handlRowsAllAttributes(rows)

	return usersByUserNameSlice, nil
}

//---------------------------------------
// облегченные (не все аттрибуты)
const commonQueryLightVersionAttributes = "select usr.user_guid, usr.user_name, usr.user_id, usr.email, " +
	" empl.employee_id, empl.employee_tabno, empl.employment, empl.employee_adress, " +
	" dep.zup_id, dep.departament_descr, " +
	" pos.position_descr, " +
	" emplCS.state_descr, emplCS.state_date_from " +
	"    from users usr " +
	"        left join employees empl on usr.user_guid=empl.employee_user " +
	"        left join departaments dep on empl.employee_departament=dep.departament_guid " +
	"        left join positions pos on pos.employee_guid=empl.employee_guid" +
	"        left join employee_states emplCS on emplCS.employee_guid=empl.employee_guid "

func handlRowsLightVersionAttributes(rows pgx.Rows) []domain.User {
	allUsersSlice := make([]domain.User, 0, 1000)
	curUser := new(domain.User)
	var oldUser = domain.User{}
	empSlice := make([]domain.Employee, 0, 2)

	for rows.Next() {

		curEmpl := new(domain.Employee)
		curDep := new(domain.Departament)
		curPos := new(domain.Position)
		curEmplSt := new(domain.EmplCurrentState)
		rows.Scan(&curUser.UserGUID, &curUser.UserName, &curUser.UserID, &curUser.UserEmail,
			&curEmpl.EmployeeId, &curEmpl.EmpTabNumber, &curEmpl.Employment, &curEmpl.EmployeeAdress,
			&curDep.DepartamentIdZUP, &curDep.DepartamentDescr,
			&curPos.PositionDescr,
			&curEmplSt.StateName, &curEmplSt.DateFrom)

		curEmpl.EmployeeDepartament = *curDep
		curEmpl.EmployeePosition = *curPos
		curEmpl.EmployeeCurrentState = *curEmplSt

		if (oldUser.UserID != "") && (curUser.UserID != oldUser.UserID) {
			oldUser.Employees = empSlice
			allUsersSlice = append(allUsersSlice, oldUser)
			empSlice = make([]domain.Employee, 0, 2)
		}

		empSlice = append(empSlice, *curEmpl)
		oldUser = *curUser
	}
	oldUser.Employees = empSlice
	allUsersSlice = append(allUsersSlice, oldUser)

	return allUsersSlice
}

// то же самое, только в map-у (по user_guid)
func handlRowsLightVersionAttributesToMap(rows pgx.Rows) map[string]domain.User {
	allUsersMap := make(map[string]domain.User)
	curUser := new(domain.User)
	var oldUser = domain.User{}
	empSlice := make([]domain.Employee, 0, 2)

	for rows.Next() {

		curEmpl := new(domain.Employee)
		curDep := new(domain.Departament)
		curPos := new(domain.Position)
		curEmplSt := new(domain.EmplCurrentState)
		rows.Scan(&curUser.UserGUID, &curUser.UserName, &curUser.UserID, &curUser.UserEmail,
			&curEmpl.EmployeeId, &curEmpl.EmpTabNumber, &curEmpl.Employment, &curEmpl.EmployeeAdress,
			&curDep.DepartamentIdZUP, &curDep.DepartamentDescr,
			&curPos.PositionDescr,
			&curEmplSt.StateName, &curEmplSt.DateFrom)

		curEmpl.EmployeeDepartament = *curDep
		curEmpl.EmployeePosition = *curPos
		curEmpl.EmployeeCurrentState = *curEmplSt

		if (oldUser.UserID != "") && (curUser.UserID != oldUser.UserID) {
			oldUser.Employees = empSlice
			allUsersMap[oldUser.UserGUID] = oldUser
			empSlice = make([]domain.Employee, 0, 2)
		}

		empSlice = append(empSlice, *curEmpl)
		oldUser = *curUser
	}
	oldUser.Employees = empSlice
	allUsersMap[oldUser.UserGUID] = oldUser

	return allUsersMap
}

// вернём всех работающих пользователей (физ.лиц) облегчённые атрибуты:
func (i *Instance) GetAllActualUsersLightVersionAttributes() ([]domain.User, error) {

	const usrs_query = commonQueryLightVersionAttributes + " where state_descr ilike $1;"

	allUsersSlice := make([]domain.User, 0)

	rows, err := i.Db.Query(context.Background(), usrs_query, "%Работ%")
	if err == pgx.ErrNoRows {
		fmt.Println("No rows")
		return allUsersSlice, err
	} else if err != nil {
		fmt.Println(err)
		return allUsersSlice, err
	}
	defer rows.Close()

	allUsersSlice = handlRowsLightVersionAttributes(rows)

	return allUsersSlice, nil
}

// вернём всех работающих пользователей (физ.лиц), у которых есть email-ы (облегчённые аттрибуты):
func (i *Instance) GetAllActualEmailUsersLightVersionAttributes() ([]domain.User, error) {

	const usrs_query = commonQueryLightVersionAttributes + " where state_descr ilike $1 and email<>'';"

	allUsersSlice := make([]domain.User, 0)

	rows, err := i.Db.Query(context.Background(), usrs_query, "%Работ%")
	if err == pgx.ErrNoRows {
		fmt.Println("No rows")
		return allUsersSlice, err
	} else if err != nil {
		fmt.Println(err)
		return allUsersSlice, err
	}
	defer rows.Close()

	allUsersSlice = handlRowsLightVersionAttributes(rows)

	return allUsersSlice, nil
}

// вернём массив только работающих пользователей с подходящими кодами - табельными номерами (?tabno=8337) облегчённые атрибуты:
func (i *Instance) GetActualUsersByTabNoLightVersionAttributes(tabno string) ([]domain.User, error) {

	const usrs_query = commonQueryLightVersionAttributes + " where user_id ilike $1 and state_descr ilike $2;"

	usersByTabNoSlice := make([]domain.User, 0)

	rows, err := i.Db.Query(context.Background(), usrs_query, "%"+tabno+"%", "%Работ%")
	if err == pgx.ErrNoRows {
		fmt.Println("No rows")
		return usersByTabNoSlice, err
	} else if err != nil {
		fmt.Println(err)
		return usersByTabNoSlice, err
	}
	defer rows.Close()

	usersByTabNoSlice = handlRowsLightVersionAttributes(rows)

	return usersByTabNoSlice, nil
}

// вернём массив только работающих пользователей с подходящими ФИО (?name=захаро) облегчённые атрибуты:
func (i *Instance) GetActualUsersByUserNameLightVersionAttributes(userName string) ([]domain.User, error) {

	const usrs_query = commonQueryLightVersionAttributes + " where user_name ilike $1 and state_descr ilike $2;"

	usersByUserNameSlice := make([]domain.User, 0)

	rows, err := i.Db.Query(context.Background(), usrs_query, "%"+userName+"%", "%Работ%")
	if err == pgx.ErrNoRows {
		fmt.Println("No rows")
		return usersByUserNameSlice, err
	} else if err != nil {
		fmt.Println(err)
		return usersByUserNameSlice, err
	}
	defer rows.Close()

	usersByUserNameSlice = handlRowsLightVersionAttributes(rows)

	return usersByUserNameSlice, nil
}

//***************************************************************************************

//***************************************************************************************

// Сотрудники

// получим массив всех сотрудников пользователя вместе с атрибутами
func selectEmployeesWithAttribsByUserGUID(insDB *pgxpool.Pool, parentUserGUID string) ([]domain.Employee, error) {

	empSlice := make([]domain.Employee, 0, 2)

	rows, err := insDB.Query(context.Background(),
		"select employee_guid from employees where employee_user=$1;",
		parentUserGUID)

	if err == pgx.ErrNoRows {
		err = fmt.Errorf("no rows: repository.SelectEmployeeByGUID error: %v", err) // "no rows" - не удалять - по ним дальше определяем добавление
		log.Error(err.Error())
		return empSlice, err
	} else if err != nil {
		log.Error("repository.SelectEmployeeByGUID error: %v", err)
		return empSlice, err
	}
	defer rows.Close()

	// такое тоже бывает, не определилось на стадии pgx.ErrNoRows
	for rows.Next() {
		curEmployee := new(domain.Employee)
		rows.Scan(&curEmployee.EmployeeGUID)
		emp, err := selectSingleEmployeeWithAttribsByGUID(insDB, curEmployee.EmployeeGUID)
		if err != nil {
			continue
		}
		empSlice = append(empSlice, *emp)
	}
	return empSlice, nil
}

func (i *Instance) SelectSingleEmployeeWithAttribsByGUID(empGUID string) (*domain.Employee, error) {
	empl, err := selectSingleEmployeeWithAttribsByGUID(i.Db, empGUID)
	if err != nil {
		return new(domain.Employee), err
	}
	return empl, nil
}

func selectSingleEmployeeWithAttribsByGUID(insDB *pgxpool.Pool, empGUID string) (*domain.Employee, error) {

	var emp domain.Employee = domain.Employee{}

	const empl_query = "select employee_guid, employee_id, employee_tabno, employee_adress, employment, employee_departament from employees where employee_guid=$1;"

	rows, err := insDB.Query(context.Background(), empl_query, empGUID)

	if err == pgx.ErrNoRows {
		err = fmt.Errorf("no rows: repository.SelectSingleEmployeeWithAttribsByGUID error: %v", err) // "no rows" - не удалять - по ним дальше определяем добавление
		log.Error(err.Error())
		return &emp, err
	} else if err != nil {
		log.Error("repository.SelectSingleEmployeeWithAttribsByGUID error: %v", err)
		return &emp, err
	}
	defer rows.Close()

	// такое тоже бывает, не определилось на стадии pgx.ErrNoRows
	if !rows.Next() {
		err = errors.New("repository.SelectSingleEmployeeWithAttribsByGUID error: no rows") // "no rows" - не удалять - по ним дальше определяем добавление
		log.Error("%v", err)
		return &emp, err
	} else {
		curEmployee := new(domain.Employee)
		rows.Scan(&curEmployee.EmployeeGUID,
			&curEmployee.EmployeeId,
			&curEmployee.EmpTabNumber,
			&curEmployee.EmployeeAdress,
			&curEmployee.Employment,
			&curEmployee.EmployeeDepartament.DepartamentGUID)
		emp = *curEmployee
	}
	// departament:
	dep, err := selectDepartamentByGUID(insDB, emp.EmployeeDepartament.DepartamentGUID)
	if err != nil {
		emp.EmployeeDepartament = domain.Departament{}
	}
	emp.EmployeeDepartament = *dep
	// current employee state:
	emplCS, err := SelectEmployeeStateByEmplGUID(insDB, emp.EmployeeGUID)
	if err != nil {
		emp.EmployeeCurrentState = domain.EmplCurrentState{}
	}
	// current employee state:
	emplPos, err := getEmplPositionByEmplGUID(insDB, emp.EmployeeGUID)
	if err != nil {
		emp.EmployeePosition = domain.Position{}
	}
	emp.EmployeeCurrentState = *emplCS
	emp.EmployeePosition = *emplPos

	return &emp, nil
}

func (i *Instance) AddEmployeeToDB(parentUserGUID string, empl *domain.Employee) (string, error) {

	commandTag, err := i.Db.Exec(context.Background(),
		"INSERT INTO employees (employee_guid, employee_user, employee_id, employee_tabno, employee_adress, employment, employee_departament) VALUES ($1, $2, $3, $4, $5, $6, $7);",
		empl.EmployeeGUID,
		parentUserGUID,
		strings.TrimSpace(empl.EmployeeId),
		strings.TrimSpace(empl.EmpTabNumber),
		strings.TrimSpace(empl.EmployeeAdress),
		strings.TrimSpace(empl.Employment),
		empl.EmployeeDepartament.DepartamentGUID)

	if err != nil {
		log.Error("repository.AddEmployeeToDB error %v", err)
		return commandTag.String(), err
	}

	return commandTag.String(), nil
}

func (i *Instance) UpdateEmployeeInDB(parentUserGUID string, empl *domain.Employee) (string, error) {

	commandTag, err := i.Db.Exec(context.Background(),
		"UPDATE employees set employee_id=$1, employee_tabno=$2, employee_adress=$3, employment=$4, employee_departament=$5 where employee_user=$6 and employee_guid=$7;",
		strings.TrimSpace(empl.EmployeeId),
		strings.TrimSpace(empl.EmpTabNumber),
		strings.TrimSpace(empl.EmployeeAdress),
		strings.TrimSpace(empl.Employment),
		empl.EmployeeDepartament.DepartamentGUID,
		parentUserGUID,
		empl.EmployeeGUID)

	if err != nil {
		log.Error("repository.UpdateEmployeeInDB error %v", err)
		return commandTag.String(), err
	}

	return commandTag.String(), nil
}

//***************************************************************************************
// Employee states
func (i *Instance) SelectEmployeeStateByEmplGUID(emplGUID string) (*domain.EmplCurrentState, error) {
	emplCS, err := SelectEmployeeStateByEmplGUID(i.Db, emplGUID)
	if err != nil {
		return new(domain.EmplCurrentState), err
	}
	return emplCS, nil
}
func SelectEmployeeStateByEmplGUID(insDB *pgxpool.Pool, emplGUID string) (*domain.EmplCurrentState, error) {
	var empCS domain.EmplCurrentState = domain.EmplCurrentState{}

	const emplCS_select_query = "select state_descr, state_date_from from employee_states where employee_guid=$1;"

	rows, err := insDB.Query(context.Background(), emplCS_select_query, emplGUID)

	if err == pgx.ErrNoRows {
		err = fmt.Errorf("no rows: repository.SelectEmployeeStateByEmplGUID error: %v", err) // "no rows" - не удалять - по ним дальше определяем добавление
		log.Error(err.Error())
		return &empCS, err
	} else if err != nil {
		log.Error("repository.SelectEmployeeStateByEmplGUID error: %v", err)
		return &empCS, err
	}
	defer rows.Close()

	// такое тоже бывает, не определилось на стадии pgx.ErrNoRows
	if !rows.Next() {
		err = errors.New("repository.SelectEmployeeStateByEmplGUID error: no rows") // "no rows" - не удалять - по ним дальше определяем добавление
		log.Error("%v", err)
		return &empCS, err
	} else {
		curEmplState := new(domain.EmplCurrentState)
		rows.Scan(&curEmplState.StateName, &curEmplState.DateFrom)
		empCS = *curEmplState
	}
	return &empCS, nil
}

func (i *Instance) AddEmployeeStateToDB(empl *domain.Employee) (string, error) {
	const emplCS_insert_query = "INSERT INTO employee_states (employee_guid, state_descr, state_date_from) VALUES ($1, $2, $3);"

	commandTag, err := i.Db.Exec(context.Background(), emplCS_insert_query,
		empl.EmployeeGUID,
		strings.TrimSpace(empl.EmployeeCurrentState.StateName),
		empl.EmployeeCurrentState.DateFrom.Format("2006-01-02"))
	if err != nil {
		log.Error("repository.AddEmployeeStateToDB error %v", err)
		return commandTag.String(), err
	}
	return commandTag.String(), nil
}

func (i *Instance) UpdateEmployeeStateToDB(empl *domain.Employee) (string, error) {
	const emplCS_update_query = "UPDATE employee_states set state_descr=$1, state_date_from=$2 where employee_guid=$3;"

	commandTag, err := i.Db.Exec(context.Background(), emplCS_update_query,
		strings.TrimSpace(empl.EmployeeCurrentState.StateName),
		empl.EmployeeCurrentState.DateFrom.Format("2006-01-02"),
		empl.EmployeeGUID)
	if err != nil {
		log.Error("repository.UpdateEmployeeStateToDB error %v", err)
		return commandTag.String(), err
	}
	return commandTag.String(), nil
}

//***************************************************************************************
// Positions
func (i *Instance) GetEmplPositionByEmplGUID(emplGUID string) (*domain.Position, error) {
	emplPos, err := getEmplPositionByEmplGUID(i.Db, emplGUID)
	if err != nil {
		return new(domain.Position), err
	}
	return emplPos, nil
}
func getEmplPositionByEmplGUID(insDB *pgxpool.Pool, emplGUID string) (*domain.Position, error) {
	var emplPos domain.Position = domain.Position{}

	const emplPos_select_query = "select position_guid, position_descr from positions where employee_guid=$1;"

	rows, err := insDB.Query(context.Background(), emplPos_select_query, emplGUID)

	if err == pgx.ErrNoRows {
		err = fmt.Errorf("no rows: repository.getEmplPositionByEmplGUID error: %v", err) // "no rows" - не удалять - по ним дальше определяем добавление
		log.Error(err.Error())
		return &emplPos, err
	} else if err != nil {
		log.Error("repository.getEmplPositionByEmplGUID error: %v", err)
		return &emplPos, err
	}
	defer rows.Close()

	// такое тоже бывает, не определилось на стадии pgx.ErrNoRows
	if !rows.Next() {
		err = errors.New("repository.getEmplPositionByEmplGUID error: no rows") // "no rows" - не удалять - по ним дальше определяем добавление
		log.Error("%v", err)
		return &emplPos, err
	} else {
		curEmplPos := new(domain.Position)
		rows.Scan(&curEmplPos.PositionGUID, &curEmplPos.PositionDescr)
		emplPos = *curEmplPos
	}
	return &emplPos, nil
}

func (i *Instance) AddEmplPositionToDB(empl *domain.Employee) (string, error) {
	const emplPos_insert_query = "INSERT INTO positions (employee_guid, position_guid, position_descr) VALUES ($1, $2, $3);"

	commandTag, err := i.Db.Exec(context.Background(), emplPos_insert_query,
		empl.EmployeeGUID,
		empl.EmployeePosition.PositionGUID,
		strings.TrimSpace(empl.EmployeePosition.PositionDescr))
	if err != nil {
		log.Error("repository.AddEmplPositionToDB error %v", err)
		return commandTag.String(), err
	}
	return commandTag.String(), nil
}

func (i *Instance) UpdateEmplPositionToDB(empl *domain.Employee) (string, error) {
	const emplPos_update_query = "UPDATE positions set position_guid=$1, position_descr=$2 where employee_guid=$3;"

	commandTag, err := i.Db.Exec(context.Background(), emplPos_update_query,
		empl.EmployeePosition.PositionGUID,
		strings.TrimSpace(empl.EmployeePosition.PositionDescr),
		empl.EmployeeGUID)
	if err != nil {
		log.Error("repository.UpdateEmplPositionToDB error %v", err)
		return commandTag.String(), err
	}
	return commandTag.String(), nil
}

func (i *Instance) DeleteEmplPositionFromDB(oldPositionGUID string) (string, error) {
	const emplPos_delete_query = "DELETE FROM positions where position_guid=$1;"

	commandTag, err := i.Db.Exec(context.Background(), emplPos_delete_query, oldPositionGUID)
	if err != nil {
		log.Error("repository.DeleteEmplPositionFromDB error %v", err)
		return commandTag.String(), err
	}
	return commandTag.String(), nil
}

//***************************************************************************************
// pshr_list - позиции штатного расписания
func (i *Instance) GetEmplPshrByEmplGUID(emplGUID string) (*domain.Pshr, error) {
	emplPshr, err := getEmplPshrByEmplGUID(i.Db, emplGUID)
	if err != nil {
		return new(domain.Pshr), err
	}
	return emplPshr, nil
}
func getEmplPshrByEmplGUID(insDB *pgxpool.Pool, emplGUID string) (*domain.Pshr, error) {
	var emplPshr domain.Pshr = domain.Pshr{}

	const emplPshr_select_query = "select pshr_guid, pshr_id, pshr_descr from pshr_list where employee_guid=$1;"

	rows, err := insDB.Query(context.Background(), emplPshr_select_query, emplGUID)

	if err == pgx.ErrNoRows {
		err = fmt.Errorf("no rows: repository.getEmplPshrByEmplGUID error: %v", err) // "no rows" - не удалять - по ним дальше определяем добавление
		log.Error(err.Error())
		return &emplPshr, err
	} else if err != nil {
		log.Error("repository.getEmplPshrByEmplGUID error: %v", err)
		return &emplPshr, err
	}
	defer rows.Close()

	// такое тоже бывает, не определилось на стадии pgx.ErrNoRows
	if !rows.Next() {
		err = errors.New("repository.getEmplPshrByEmplGUID error: no rows") // "no rows" - не удалять - по ним дальше определяем добавление
		log.Error("%v", err)
		return &emplPshr, err
	} else {
		curEmplPshr := new(domain.Pshr)
		rows.Scan(&curEmplPshr.PshrGUID, &curEmplPshr.PshrId, &curEmplPshr.PshrDescr)
		emplPshr = *curEmplPshr
	}
	return &emplPshr, nil
}

func (i *Instance) AddEmplPshrToDB(empl *domain.Employee) (string, error) {
	const emplPshr_insert_query = "INSERT INTO pshr_list (employee_guid, pshr_guid, pshr_id, pshr_descr) VALUES ($1, $2, $3, $4);"

	commandTag, err := i.Db.Exec(context.Background(), emplPshr_insert_query,
		empl.EmployeeGUID,
		empl.EmployeePshr.PshrGUID,
		strings.TrimSpace(empl.EmployeePshr.PshrId),
		strings.TrimSpace(empl.EmployeePshr.PshrDescr))
	if err != nil {
		log.Error("repository.AddEmplPshrToDB error %v", err)
		return commandTag.String(), err
	}
	return commandTag.String(), nil
}

func (i *Instance) UpdateEmplPshrToDB(empl *domain.Employee) (string, error) {
	const emplPshr_update_query = "UPDATE pshr_list set pshr_guid=$1, pshr_id=$2, pshr_descr=$3 where employee_guid=$4;"

	commandTag, err := i.Db.Exec(context.Background(), emplPshr_update_query,
		empl.EmployeePshr.PshrGUID,
		strings.TrimSpace(empl.EmployeePshr.PshrId),
		strings.TrimSpace(empl.EmployeePshr.PshrDescr),
		empl.EmployeeGUID)
	if err != nil {
		log.Error("repository.UpdateEmplPshrToDB error %v", err)
		return commandTag.String(), err
	}
	return commandTag.String(), nil
}

func (i *Instance) DeleteEmplPshrFromDB(oldPshrGUID string) (string, error) {
	const emplPos_delete_query = "DELETE FROM pshr_list where pshr_guid=$1;"

	commandTag, err := i.Db.Exec(context.Background(), emplPos_delete_query, oldPshrGUID)
	if err != nil {
		log.Error("repository.DeleteEmplPshrFromDB error %v", err)
		return commandTag.String(), err
	}
	return commandTag.String(), nil
}

//***************************************************************************************
// Подразделения

// Получим подразделение по GUID
func (i *Instance) SelectDepByGUID(DepGUID string) (*domain.Departament, error) {
	dep, err := selectDepartamentByGUID(i.Db, DepGUID)
	if err != nil {
		return new(domain.Departament), err
	}
	return dep, nil
}

func selectDepartamentByGUID(insDB *pgxpool.Pool, DepGUID string) (*domain.Departament, error) {
	var dep *domain.Departament = new(domain.Departament)

	rows, err := insDB.Query(context.Background(),
		"select departament_guid, departament_descr, departament_parent_guid, zup_id, zup_parent_id, zup_not_used_from from departaments where departament_guid=$1;",
		DepGUID)

	if err == pgx.ErrNoRows {
		err = fmt.Errorf("no rows: repository.selectDepartamentsByGUID error: %v", err) // "no rows" - не удалять - по ним дальше определяем добавление
		log.Error(err.Error())
		return dep, err // 0
	} else if err != nil {
		log.Error("repository.selectDepartamentsByGUID error: %v", err)
		return dep, err
	}
	defer rows.Close()

	// такое тоже бывает, не определилось на стадии pgx.ErrNoRows
	if !rows.Next() {
		err = fmt.Errorf("no rows: repository.selectDepartamentsByGUID error: %v", err) // "no rows" - не удалять - по ним дальше определяем добавление
		log.Error(err.Error())
		return dep, err // 0
	} else {
		curDep := new(domain.Departament)
		rows.Scan(&curDep.DepartamentGUID,
			&curDep.DepartamentDescr,
			&curDep.DepartamentParentGUID,
			&curDep.DepartamentIdZUP,
			&curDep.DepartamentParentIdZUP,
			&curDep.DepartamentNotUsedFrom)
		dep = curDep
	}

	return dep, nil
}

// Найти guid родителя по его ParentIdZUP
// (Есть объект. У него есть родитель с ParentIdZUP. Нужно найти guid родителя, чтобы записать его в departament_parent_guid)
// внутренняя функция, используется внутри пакета
func getDepParentGUIDByParentIdZUP(insDB *pgxpool.Pool, parentIdZUP string) (string, error) {
	var parentDepGUID string

	rows, err := insDB.Query(context.Background(), "select departament_guid from departaments where zup_parent_id=$1;", parentIdZUP)
	if err == pgx.ErrNoRows {
		err = fmt.Errorf("no rows: repository.getDepParentGUIDByParentIdZUP error: %v", err) // "no rows" - не удалять - по ним дальше определяем добавление
		log.Error(err.Error())
		return parentDepGUID, err
	} else if err != nil {
		log.Error("repository.getDepParentGUIDByParentIdZUP error: %v", err)
		return parentDepGUID, err
	}
	defer rows.Close()

	if !rows.Next() {
		err = errors.New("repository.getDepParentGUIDByParentIdZUP error: no rows") // "no rows" - не удалять - по ним дальше определяем добавление
		log.Error("%v", err)
		return parentDepGUID, err
	} else {
		rows.Scan(&parentDepGUID)
		if strings.TrimSpace(parentDepGUID) != "" {
			return parentDepGUID, nil
		}
	}

	return parentDepGUID, nil // видимо, просто нет записи о родителе
}

func (i *Instance) AddDepartament(dep *domain.Departament) (string, error) {
	var curDepParentGUID string
	if strings.TrimSpace(dep.DepartamentParentGUID) == "" {
		curDepParentGUID, _ = getDepParentGUIDByParentIdZUP(i.Db, dep.DepartamentParentIdZUP) // получим guid родителя, если он есть
	} else {
		curDepParentGUID = dep.DepartamentParentGUID
	}

	commandTag, err := i.Db.Exec(context.Background(),
		"INSERT INTO departaments (departament_guid, departament_descr, departament_parent_guid, zup_id, zup_parent_id, zup_not_used_from) VALUES ($1, $2, $3, $4, $5, $6);",
		dep.DepartamentGUID,
		strings.TrimSpace(dep.DepartamentDescr),
		curDepParentGUID,
		strings.TrimSpace(dep.DepartamentIdZUP),
		strings.TrimSpace(dep.DepartamentParentIdZUP),
		dep.DepartamentNotUsedFrom.Format("2006-01-02"))
	if err != nil {
		log.Error("repository.AddDepartament error %v", err)
		return commandTag.String(), err
	}

	return commandTag.String(), nil
}

func (i *Instance) UpdateDepartament(dep *domain.Departament) (string, error) {
	curDepParentGUID, _ := getDepParentGUIDByParentIdZUP(i.Db, dep.DepartamentParentIdZUP) // получим guid родителя, если он есть

	commandTag, err := i.Db.Exec(context.Background(),
		"UPDATE departaments set departament_descr=$1, departament_parent_guid=$2, zup_id=$3, zup_parent_id=$4, zup_not_used_from=$5 where departament_guid=$6;",
		strings.TrimSpace(dep.DepartamentDescr),
		curDepParentGUID,
		strings.TrimSpace(dep.DepartamentIdZUP),
		strings.TrimSpace(dep.DepartamentParentIdZUP),
		dep.DepartamentNotUsedFrom.Format("2006-01-02"),
		dep.DepartamentGUID)
	if err != nil {
		log.Error("repository.UpdateDepartament error %v", err)
		return commandTag.String(), err
	}

	return commandTag.String(), nil
}

// вернём все подразделения (и активные, и расформированные):
func (i *Instance) GetAllDepartaments() ([]domain.Departament, error) {

	const departaments_query = "select departament_guid, zup_id, departament_descr, zup_parent_id, zup_not_used_from, departament_parent_guid from departaments;"

	allDepartamentsSlice := make([]domain.Departament, 0)

	rows, err := i.Db.Query(context.Background(), departaments_query)
	if err == pgx.ErrNoRows {
		fmt.Println("No rows")
		return allDepartamentsSlice, err
	} else if err != nil {
		fmt.Println(err)
		return allDepartamentsSlice, err
	}
	defer rows.Close()

	for rows.Next() {
		curDep := new(domain.Departament)
		rows.Scan(
			&curDep.DepartamentGUID,
			&curDep.DepartamentIdZUP,
			&curDep.DepartamentDescr,
			&curDep.DepartamentParentIdZUP,
			&curDep.DepartamentNotUsedFrom,
			&curDep.DepartamentParentGUID)

		allDepartamentsSlice = append(allDepartamentsSlice, *curDep)
	}

	return allDepartamentsSlice, nil
}

// вернём только все актуальные подразделения (не расформированные):
func (i *Instance) GetActualDepartaments() ([]domain.Departament, error) {

	const departaments_query = "select departament_guid, zup_id, departament_descr, zup_parent_id, zup_not_used_from, departament_parent_guid " +
		" from departaments where zup_parent_id <> '000999999' and zup_not_used_from = '0001-01-01';"

	allDepartamentsSlice := make([]domain.Departament, 0)

	rows, err := i.Db.Query(context.Background(), departaments_query)
	if err == pgx.ErrNoRows {
		fmt.Println("No rows")
		return allDepartamentsSlice, err
	} else if err != nil {
		fmt.Println(err)
		return allDepartamentsSlice, err
	}
	defer rows.Close()

	for rows.Next() {
		curDep := new(domain.Departament)
		rows.Scan(
			&curDep.DepartamentGUID,
			&curDep.DepartamentIdZUP,
			&curDep.DepartamentDescr,
			&curDep.DepartamentParentIdZUP,
			&curDep.DepartamentNotUsedFrom,
			&curDep.DepartamentParentGUID)

		allDepartamentsSlice = append(allDepartamentsSlice, *curDep)
	}

	return allDepartamentsSlice, nil
}

// вернём все актуальные подразделения по родителю:
func (i *Instance) GetActualDepartamentsByParentIdZUP(parentIdZUP string) ([]domain.Departament, error) {

	const departaments_query = "select departament_guid, zup_id, departament_descr, zup_parent_id, zup_not_used_from, departament_parent_guid " +
		" from departaments where zup_parent_id <> '000999999' and zup_not_used_from = '0001-01-01' and zup_parent_id=$1;"

	allDepartamentsSlice := make([]domain.Departament, 0)

	rows, err := i.Db.Query(context.Background(), departaments_query, parentIdZUP)
	if err == pgx.ErrNoRows {
		fmt.Println("No rows")
		return allDepartamentsSlice, err
	} else if err != nil {
		fmt.Println(err)
		return allDepartamentsSlice, err
	}
	defer rows.Close()

	for rows.Next() {
		curDep := new(domain.Departament)
		rows.Scan(
			&curDep.DepartamentGUID,
			&curDep.DepartamentIdZUP,
			&curDep.DepartamentDescr,
			&curDep.DepartamentParentIdZUP,
			&curDep.DepartamentNotUsedFrom,
			&curDep.DepartamentParentGUID)

		allDepartamentsSlice = append(allDepartamentsSlice, *curDep)
	}

	return allDepartamentsSlice, nil
}

//***************************************************************************************
// Обмены
//------------------------------------------------------
// Регистрация к обмену
func (i *Instance) AddToExchange(exch *domain.ExchangeStruct) (string, error) {
	const emplPos_insert_query = "INSERT INTO exchanges (base_id, r_id, rowdata, date_init) VALUES ($1, $2, $3, $4);"

	commandTag, err := i.Db.Exec(context.Background(), emplPos_insert_query,
		exch.BaseID,
		exch.ReasonID,
		strings.TrimSpace(exch.RowData),
		time.Now())
	if err != nil {
		log.Error("repository.AddToExchange error %v", err)
		return commandTag.String(), err
	}
	return commandTag.String(), nil
}

// Получим данные user-ов (физ. лиц) к обмену , у которых resp_status не равен 200(ok)
// метод основан на том, что в поле exch.rowdata не произвольные данные, а user_guid
func (i *Instance) GetAllUsersToExchange(reasonID int) ([]string, *map[int]domain.Exchange1СErrorsStruct, error) { // []domain.User

	ctx, cancel := context.WithTimeout(context.Background(), 1000*time.Millisecond)
	defer cancel()

	mapExID := map[int]domain.Exchange1СErrorsStruct{} // id строк, в которые записывать ответ из 1С (для обработки ошибок)
	// в мапу пишем id отправляемой строки и attempt_count

	const query = "select exch.ex_id, exch.attempt_count, " +
		"              usr.user_guid, usr.user_name, usr.user_id, usr.email " +
		"                  from exchanges exch " +
		"                      left join users usr on cast(exch.rowdata as uuid)=usr.user_guid " +
		"                          where (exch.r_id=$1 and exch.resp_status<>'200(ok)'); "

	//usersSlice := make([]domain.User, 0)
	usersSlice := make([]string, 0)

	rows, err := i.Db.Query(ctx, query, reasonID)
	if err == pgx.ErrNoRows {
		fmt.Println("No rows")
		return usersSlice, &mapExID, err
	} else if err != nil {
		fmt.Println(err)
		return usersSlice, &mapExID, err
	}
	defer rows.Close()

	var exID, exAttCount int
	for rows.Next() {
		curUser := new(domain.User)
		rows.Scan(&exID, &exAttCount, &curUser.UserGUID, &curUser.UserName, &curUser.UserID, &curUser.UserEmail)
		mapExID[exID] = domain.Exchange1СErrorsStruct{UserGUID: curUser.UserGUID, AttemptCount: exAttCount}
		usersSlice = append(usersSlice, *&curUser.UserGUID)
	}

	return usersSlice, &mapExID, nil
}

//------------------------------------------------------
// апдейтим записи по одной - цикл в вызывающей функции, т.к. у них может быть разное значение "attempt_count"
func (ins *Instance) SetExchangeStatus(ex_id int, attempt_count int, errorStr string) error {
	// пробуем 10 раз, если нет, то будем писать в redis и потом пробовать еще
	const update_query = "update exchanges set attempt_count=$1, last_attempt_date=$2, resp_status=$3 where ex_id=$4;"

	_, err := ins.Db.Exec(context.Background(), update_query,
		attempt_count+1,
		time.Now(),
		errorStr,
		ex_id)
	if err != nil {
		return fmt.Errorf("repository.SetExchangeStatus error: не записан результат обмена с ex_id %d по причине %v", ex_id, err)
	}
	return nil
}

//***************************************************************************************
// Рассылки
//------------------------------------------------------

// common notifications
// получить список email-ов user-ов по типу рассылки
func (ins *Instance) GetUserEmailsByNotificationsTypes(notitype int) ([]string, error) {
	const bd_usrs_query_debug_all = "select usr.email " +
		" 							 from users_for_notifications un " +
		" 								left join users usr on un.user_guid=usr.user_guid " +
		" 							 where un.notitype = $1;"

	usersEmailsSlice := make([]string, 0)

	rows, err := ins.Db.Query(context.Background(), bd_usrs_query_debug_all, notitype)
	if err == pgx.ErrNoRows {
		return usersEmailsSlice, err
	} else if err != nil {
		return usersEmailsSlice, err
	}
	defer rows.Close()

	for rows.Next() {
		var curUserEmail string
		rows.Scan(&curUserEmail)

		usersEmailsSlice = append(usersEmailsSlice, curUserEmail)
	}
	return usersEmailsSlice, nil

}

//------------------------------------------------------
// вернуть всех именинников, ДР которых приходится на конкретную дату DEBUG !!!!!!!!!!!!!!!!!!!!!!!!!
func (ins *Instance) GetBdOwnersOneDayDebug(dateTempl string) (map[string][]domain.User, error) { //([]domain.User, error) {

	const bd_usrs_query_debug_all = "select user_name, user_birthday from users " +
		"								where cast(user_birthday as text) like $1;"

	var retMap map[string][]domain.User = map[string][]domain.User{} // одному observer - у сопоставляем много bd_owner - ов
	adminSlice, err := ins.GetUserEmailsByNotificationsTypes(1)      // admin    []string{"akabanov@VODOKANAL-NN.RU"}
	if err != nil {
		return retMap, err
	}
	admin := adminSlice[0]

	rows, err := ins.Db.Query(context.Background(), bd_usrs_query_debug_all, "%"+dateTempl+"%") //  "-05-31"
	if err == pgx.ErrNoRows {
		return retMap, err
	} else if err != nil {
		return retMap, err
	}
	defer rows.Close()

	for rows.Next() {
		curBdOwner := new(domain.User)
		err = rows.Scan(&curBdOwner.UserName, &curBdOwner.UserBirthday)
		if err != nil {
			return retMap, err
		}
		curBdOwnerSlice, ok := retMap[admin]
		if !ok {
			curBdOwnerSlice = make([]domain.User, 0)
		}
		curBdOwnerSlice = append(curBdOwnerSlice, *curBdOwner)
		retMap[admin] = curBdOwnerSlice
	}

	return retMap, nil
}

// вернуть всех именинников, ДР которых приходится на конкретную дату, а также их наблюдателей
func (ins *Instance) GetBdObserversOwnersByTempl(dateTempl string) (map[string][]domain.User, error) {

	const bd_usrs_query = "select " +
		" 						usr.user_name, " +
		" 						usr.email, " + //  as observer_email
		" 						usr1.user_name, " + //  as bd_owner
		" 						usr1.user_birthday " + // as bd
		"				   from bd_notifications bd " +
		" 						inner join users usr on bd.bd_observer_guid=usr.user_guid " +
		" 						inner join users usr1 on bd.bd_owner_guid=usr1.user_guid " +
		"				   where cast(usr1.user_birthday as text) like $1;"

	// одному observer-у (key) сопоставляем много bd_owner-ов (value)
	var retMap map[string][]domain.User = map[string][]domain.User{}

	rows, err := ins.Db.Query(context.Background(), bd_usrs_query, "%"+dateTempl+"%") //  "-05-31"
	if err == pgx.ErrNoRows {
		return retMap, err
	} else if err != nil {
		return retMap, err
	}
	defer rows.Close()

	for rows.Next() {
		var curObserverUserName string
		var curObserverEmail string
		curBdOwner := new(domain.User)
		err = rows.Scan(&curObserverUserName, &curObserverEmail, &curBdOwner.UserName, &curBdOwner.UserBirthday)
		if err != nil {
			return retMap, err
		}
		if strings.TrimSpace(curObserverEmail) == "" {
			log.Error("bd_notifications repository.GetBdObserversOwnersnextWeek error: для пользователя %v не найден email.", curObserverUserName)
			continue
		}
		curBdOwnerSlice, ok := retMap[curObserverEmail]
		if !ok {
			curBdOwnerSlice = make([]domain.User, 0)
		}
		curBdOwnerSlice = append(curBdOwnerSlice, *curBdOwner)
		retMap[curObserverEmail] = curBdOwnerSlice
	}

	return retMap, nil
}

// вернуть всех именинников, ДР которых приходится на nextWeek, а также их наблюдателей
func (ins *Instance) GetBdObserversOwnersnextWeek(nextWeek domain.NextWeekStruct) (map[string][]domain.User, error) {

	const bd_usrs_query = "select " +
		" 						usr.user_name, " +
		" 						usr.email, " +
		" 						usr1.user_name, " +
		" 						usr1.user_birthday " +
		"				   from bd_notifications bd " +
		" 						inner join users usr on bd.bd_observer_guid=usr.user_guid " +
		" 						inner join users usr1 on bd.bd_owner_guid=usr1.user_guid " +
		"				   where ((cast(usr1.user_birthday as text) like $1) or " + // '%-01-17%'
		" 						  (cast(usr1.user_birthday as text) like $2) or " +
		" 						  (cast(usr1.user_birthday as text) like $3) or " +
		" 						  (cast(usr1.user_birthday as text) like $4) or " +
		" 						  (cast(usr1.user_birthday as text) like $5) or " +
		" 						  (cast(usr1.user_birthday as text) like $6) or " +
		" 						  (cast(usr1.user_birthday as text) like $7));"

	// одному observer-у (key) сопоставляем много bd_owner-ов (value)
	var retMap map[string][]domain.User = map[string][]domain.User{}

	rows, err := ins.Db.Query(context.Background(), bd_usrs_query,
		"%"+nextWeek.Monday+"%",
		"%"+nextWeek.Tuesday+"%",
		"%"+nextWeek.Wednesday+"%",
		"%"+nextWeek.Thursday+"%",
		"%"+nextWeek.Friday+"%",
		"%"+nextWeek.Saturday+"%",
		"%"+nextWeek.Sunday+"%")

	if err == pgx.ErrNoRows {
		return retMap, err
	} else if err != nil {
		return retMap, err
	}
	defer rows.Close()

	for rows.Next() {
		var curObserverUserName string
		var curObserverEmail string
		curBdOwner := new(domain.User)
		err = rows.Scan(&curObserverUserName, &curObserverEmail, &curBdOwner.UserName, &curBdOwner.UserBirthday)
		if err != nil {
			return retMap, err
		}
		if strings.TrimSpace(curObserverEmail) == "" {
			log.Error("bd_notifications repository.GetBdObserversOwnersnextWeek error: для пользователя %v не найден email.", curObserverUserName)
			continue
		}
		curBdOwnerSlice, ok := retMap[curObserverEmail]
		if !ok {
			curBdOwnerSlice = make([]domain.User, 0)
		}
		curBdOwnerSlice = append(curBdOwnerSlice, *curBdOwner)
		retMap[curObserverEmail] = curBdOwnerSlice
	}

	return retMap, nil
}

//------------------------------------------------------
// сделать запись в таблицу "напомнить кому" - "напомнить о ком"
func (ins *Instance) InsertBdObsOwners(bdObserverId, bdOwnerId string) error {
	const bd_usrs_query = "insert into bd_notifications " +
		" 				       (bd_observer_guid, bd_owner_guid) " +
		" 				   values( " +
		" 					   (select user_guid from users where user_id like $1), " +
		"					   (select user_guid from users where user_id like $2)); "

	_, err := ins.Db.Exec(context.Background(), bd_usrs_query, "%"+bdObserverId+"%", "%"+bdOwnerId+"%")
	if err == pgx.ErrNoRows {
		return err
	} else if err != nil {
		// pqErr := err.(*pq.Error)
		// if pqErr.Code.Name() == "SQLSTATE 23505" {
		// 	return fmt.Errorf("Ограничения уникальности %v", err)
		// }
		if strings.Contains(err.Error(), "SQLSTATE 23505") {
			err = fmt.Errorf("Ограничения уникальности. Пара %s и %s уже существует. %v", bdObserverId, bdOwnerId, err)
			return err
		}
		return err
	}

	return nil
}
