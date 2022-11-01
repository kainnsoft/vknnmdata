package repository

import (
	"context"
	"fmt"
	"mdata/internal/domain"
	"time"

	"github.com/jackc/pgx/v4"
)

type UsersModeller interface {
	GetUsersFiredFrom(time.Time) ([]domain.User, error)
}

// все user-ы в базе, у которых больше одного сотрудника и которые работают (хотя бы один сотрудник имеет статус "Работа")
func getSeveralEmployesActualUsers(i *PostgreInstance) (map[string]int, error) {
	//ctx, cancel := context.WithTimeout(context.Background(), queryTimeout*time.Millisecond)
	//defer cancel()
	ctx := context.Background()

	mapOfUsers := make(map[string]int)
	const severalEmployesActualUsersQuery = "select user_guid, user_id " + //, user_name, haswork, state_date_from " +
		" from( " +
		"     select usr.user_guid, max(usr.user_name) user_name, count(usr.user_id) user_id, max(emplCS.state_date_from) state_date_from, " +
		"         max(case when state_descr like '%Увольнен%' then 0 else 1 end) as haswork " +
		"     from users usr " +
		"	      left join employees empl on usr.user_guid=empl.employee_user " +
		"	      left join employee_states emplCS on emplCS.employee_guid=empl.employee_guid " +
		"     GROUP BY user_guid) foo " +
		" where haswork = 1 and user_id > 1;"

	rows, err := i.Db.Query(ctx, severalEmployesActualUsersQuery)
	if err == pgx.ErrNoRows {
		err = fmt.Errorf("repository.severalEmployesActualUsersQuery error: No rows: %v", err)
		return mapOfUsers, err
	} else if err != nil {
		return mapOfUsers, err
	}
	defer rows.Close()

	var strUserGuid string
	var emplCount int
	for rows.Next() {
		rows.Scan(&strUserGuid, &emplCount)
		mapOfUsers[strUserGuid] = emplCount
	}

	return mapOfUsers, nil
}

// вернём всех уволенных пользователей (физ.лиц) все атрибуты:
// 1) Формируем map исключений.
// 2) Получаем список уволенных (готовый код)
// 3) Исключаем из п.2 тех, кто есть в п.1
func (i *PostgreInstance) GetUsersFiredFrom(from time.Time) ([]domain.User, error) {
	//ctx, cancel := context.WithTimeout(context.Background(), queryTimeout*time.Millisecond)
	//defer cancel()
	ctx := context.Background()

	firedUsersSlice := make([]domain.User, 0)

	mapOfUsers, err := getSeveralEmployesActualUsers(i)
	if err != nil {
		return firedUsersSlice, err
	}
	const usrs_query = commonQueryLightVersionAttributes + " where state_descr ilike $1 and state_date_from >= $2;"

	rows, err := i.Db.Query(ctx, usrs_query, "%Увольнен%", from)
	if err == pgx.ErrNoRows {
		err = fmt.Errorf("repository.GetUsersFiredFrom error: No rows: %v", err)
		return firedUsersSlice, err
	} else if err != nil {
		return firedUsersSlice, err
	}
	defer rows.Close()

	// получаем полную map-у уволенных, в т.ч. тех, кто вернулся, переведен через увольнение и т.д.
	firedUsersMap := handlRowsLightVersionAttributesToMap(rows)

	// пробежим по map-е исключений (mapOfUsers) и выкинем их из полученной (firedUsersMap):
	for userGuid, _ := range mapOfUsers {
		delete(firedUsersMap, userGuid)
	}

	for _, user := range firedUsersMap {
		firedUsersSlice = append(firedUsersSlice, user)
	}

	return firedUsersSlice, nil
}
