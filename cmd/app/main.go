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
	// cfg := &domain.Config{}
	// cfg.DBHost = repository.ReadConfig("db.host")
	// cfg.DBUsername = repository.ReadConfig("db.username")
	// cfg.DBPassword = repository.ReadConfig("db.password")
	// cfg.DBPort = repository.ReadConfig("db.port")
	// cfg.DataBaseName = repository.ReadConfig("db.dbname")
	// cfg.DBTimeout, _ = strconv.Atoi(repository.ReadConfig("db.timeout"))

	//Создаем конфиг для пула
	poolConfig, err := repository.NewPoolConfig(cfg)
	if err != nil {
		log.Error("main: Pool config error:", err)
		panic(err)
	}

	//Устанавливаем максимальное количество соединений, которые могут находиться в ожидании
	poolConfig.MaxConns = 5

	//Создаем пул подключений
	pool, err := repository.NewConnection(poolConfig)
	if err != nil {
		log.Error("main: Connect to database  error:", err)
		panic(err)
	}
	fmt.Println("DB connection OK!")
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
