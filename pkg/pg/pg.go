package pg

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
)

const (
	_defaultMaxPoolSize = 5
	_defaultConnTimeout = time.Second
)

type DB struct {
	Pool *pgxpool.Pool
}

func NewInsPgDB(strurl string, maxPoolSize int) (*DB, error) {
	_ConnAttempts := 3

	insPgdb := &DB{}

	// Создаем конфиг для пула
	poolConfig, err := NewPoolConfig(strurl)
	if err != nil {
		return nil, fmt.Errorf("pool config error: %v", err)
	}

	// Устанавливаем максимальное количество соединений, которые могут находиться в ожидании
	if maxPoolSize > 1 {
		poolConfig.MaxConns = int32(maxPoolSize)
	} else {
		poolConfig.MaxConns = _defaultMaxPoolSize
	}

	// включаем передачу бинарных параметров (недокументированная фича Debug)
	poolConfig.ConnConfig.PreferSimpleProtocol = true

	var connPool *pgxpool.Pool

	for _ConnAttempts > 0 {
		connPool, err = NewConnection(poolConfig)
		if err == nil {
			break
		}

		log.Printf("Postgres is trying to connect, attempts left: %d", _ConnAttempts)
		time.Sleep(_defaultConnTimeout)
		_ConnAttempts--
	}

	if err != nil {
		return nil, fmt.Errorf("postgres - NewPostgres - connAttempts == 0: %w", err)
	}

	insPgdb.Pool = connPool

	return insPgdb, nil
}

// Config pool-а подключений
func NewPoolConfig(connStr string) (*pgxpool.Config, error) {
	poolConfig, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, err
	}

	return poolConfig, nil
}

// Функция-обертка для создания подключения с помощью пула
func NewConnection(poolConfig *pgxpool.Config) (*pgxpool.Pool, error) {
	conn, err := pgxpool.ConnectConfig(context.Background(), poolConfig)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

// Close -.
func (p *DB) Close() {
	if p.Pool != nil {
		p.Pool.Close()
	}
}
