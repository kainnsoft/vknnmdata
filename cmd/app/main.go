package main

import (
	"flag"
	"fmt"
	"net/http"

	"mdata/internal/repository"
	"mdata/internal/routes"
	log "mdata/pkg/logging"

	_ "net/http/pprof"

	"github.com/jackc/pgx/v4/pgxpool"
)

var ins *repository.Instance // db
var pool *pgxpool.Pool

func createDBConnPool() *pgxpool.Pool {
	//Задаем параметры для подключения к БД
	cfg := repository.GetCfg()

	//Создаем конфиг для пула
	poolConfig, err := repository.NewPoolConfig(cfg)
	if err != nil {
		log.Error("main: Pool config error:", err)
		panic(err)
	}

	//Устанавливаем максимальное количество соединений, которые могут находиться в ожидании
	poolConfig.MaxConns = 10
	poolConfig.ConnConfig.PreferSimpleProtocol = true // включаем передачу бинарных параметров // недокументированная фича Debug

	//Создаем пул подключений
	pool, err := repository.NewConnection(poolConfig)
	if err != nil {
		log.Error("main: Connect to database  error:", err)
		panic(err)
	}
	fmt.Println("DB connection OK!")
	fmt.Printf("Conections - Max: %d, Iddle: %d, Total: %d \n", pool.Stat().MaxConns(), pool.Stat().IdleConns(), pool.Stat().TotalConns())
	return pool
}

func openDB() {
	// Откроем базу данных:
	pool = createDBConnPool()
	ins = &repository.Instance{Db: pool}
}

func main() {

	// Откроем базу данных:
	if ins == nil {
		openDB()
	}
	defer pool.Close()

	// server   --------------------------------------------------------
	mux := http.NewServeMux()

	routes.InitializeRoutes(mux, ins)

	addr := flag.String("addr", ":8080", "Сетевой адрес веб-сервера MD")
	flag.Parse()

	err := http.ListenAndServe(*addr, mux)
	log.Error("main: srv.ListenAndServe() error: %v", err)
}
