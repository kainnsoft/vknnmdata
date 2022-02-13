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

// вернём всех уволенных пользователей (физ.лиц) все атрибуты:
func (i *Instance) GetUsersFiredFrom(from time.Time) ([]domain.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), queryTimeout*time.Millisecond)
	defer cancel()

	//const usrs_query = commonQueryAllAttributes + " where state_descr ilike $1;"
	const usrs_query = commonQueryLightVersionAttributes + " where state_descr ilike $1 and state_date_from >= $2;"

	firedUsersSlice := make([]domain.User, 0)

	rows, err := i.Db.Query(ctx, usrs_query, "%Увольнен%", from)
	if err == pgx.ErrNoRows {
		fmt.Println("No rows")
		return firedUsersSlice, err
	} else if err != nil {
		fmt.Println(err)
		return firedUsersSlice, err
	}
	defer rows.Close()

	firedUsersSlice = handlRowsLightVersionAttributes(rows)

	return firedUsersSlice, nil
}
