package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"mdata/internal/domain"
	"mdata/internal/handlers"
	log "mdata/internal/logging"
	"mdata/internal/repository"

	_ "net/http/pprof"

	"github.com/jackc/pgx/v4/pgxpool"
)

var ins *repository.Instance // db
var pool *pgxpool.Pool

func createDBConnPool() *pgxpool.Pool {
	//Задаем параметры для подключения к БД
	cfg := &domain.Config{}
	cfg.DBHost = repository.ReadConfig("db.host")
	cfg.DBUsername = repository.ReadConfig("db.username")
	cfg.DBPassword = repository.ReadConfig("db.password")
	cfg.DBPort = repository.ReadConfig("db.port")
	cfg.DataBaseName = repository.ReadConfig("db.dbname")
	cfg.DBTimeout, _ = strconv.Atoi(repository.ReadConfig("db.timeout"))

	//Создаем конфиг для пула
	poolConfig, err := repository.NewPoolConfig(cfg)
	if err != nil {
		log.Error("main: Pool config error:", err)
		os.Exit(1)
		panic(err)
	}

	//Устанавливаем максимальное количество соединений, которые могут находиться в ожидании
	poolConfig.MaxConns = 5

	//Создаем пул подключений
	pool, err := repository.NewConnection(poolConfig)
	if err != nil {
		log.Error("main: Connect to database  error:", err)
		os.Exit(1)
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

	// ping - метод. Получим доступность базы 1С:ЗУП
	http.HandleFunc("/from-zup/ping/", handlers.PingZup)

	// ping - метод. Получим всех пользователей 1С:ЗУП
	http.HandleFunc("/from-zup/ping/AllUsers/", handlers.PingFromZmupAllUsers)

	// Запишем в БД всех user-ов из ЗУП-а, по пути получив их email из AD                     (daily task)
	http.HandleFunc("/db/from-zup/write/all-users/", handlers.RestHandleZupWriteAllUsers(ins))

	// Debug-method. Запишем в БД одного user-а из запроса (из Postman-а), по пути получив его email из AD
	http.HandleFunc("/db/from-zup/write/one-user/", handlers.RestHandleDebugWriteOneUser(ins))

	// Debug-method Отправляем в 1С:ЗУП email одного захардкоженного пользователя.
	http.HandleFunc("/to-zup/send-email/ping/", handlers.RestPutOneUserEmailToZup)

	// Отправляем в 1С:ЗУП email всех пользователей для обновления email (можно один, но в массиве)    (daily task)
	http.HandleFunc("/to-zup/send-email/put-array/", handlers.RestPutAllUsersWithEmailToZup(ins))

	// Отправляем в 1С:CreateUser даные новых сотрудников (users) для создания для них пользователей в 1С (можно один, но в массиве)    (daily task)
	http.HandleFunc("/to-1cuser/send-accounts-crud/put-array/", handlers.RestPutNewUsersWithEmailToCreate1CAccounts(ins))

	// блок "Подразделения"
	// ping-метод - получить все подразделения из 1С:ЗУП
	http.HandleFunc("/from-zup/ping/AllDepartaments/", handlers.RestGetFromZupPingAllDepartaments())

	// получить все подразделения из 1С:ЗУП и записать их в базу MD
	http.HandleFunc("/db/from-zup/write/all-departaments/", handlers.RestHandleFromZupAllDepartament(ins))

	// Debug-method. Запишем в БД одно подразделение из запроса (из Postman-а)
	http.HandleFunc("/db/from-zup/write/singl-departament/", handlers.RestHandleDebugWriteSingleDepartament(ins))

	//------------------------------------------------------------------
	// блок REST api  --------------------------------------------------
	//------------------------------------------------------------------
	// ping - метод. Доступность сервиса MD
	http.HandleFunc("/", handlers.PingMasterData)

	// ping - метод. Доступность DB
	http.HandleFunc("/db/ping/", handlers.PingDB(ins))
	//------------------------------------------------------------------
	// все аттрибуты
	// отдать всех работающих (актуальных) физ.лиц (все аттрибуты)
	http.HandleFunc("/get-act-employees/", handlers.RestSendAllEmployeesAllAttributes(ins))

	// отдать всех работающих (актуальных) физ.лиц, у которых есть email-ы (все аттрибуты)
	http.HandleFunc("/get-act-email-employees/", handlers.RestSendAllEmailEmployeesAllAttributes(ins))

	// отдать сотрудника(-ков) работающих (актуальных) по параметрам ?tabno=8337 или ?name=Кабанов (обрабатывается по принципу like) (все аттрибуты)
	http.HandleFunc("/get-act-employee", handlers.RestSendEmployeeAllAttributes(ins))

	//------------------------------------------------------------------
	// облегчённые аттрибуты
	// отдать всех работающих (актуальных) физ.лиц (облегчённые аттрибуты)
	http.HandleFunc("/get-act-employees-light/", handlers.RestSendAllEmployeesLightVersionAttributes(ins))

	// отдать всех работающих (актуальных) физ.лиц, у которых есть email-ы (облегчённые аттрибуты)
	http.HandleFunc("/get-act-email-employees-light/", handlers.RestSendAllEmailEmployeesLightVersionAttributes(ins))

	// отдать сотрудника(-ков) работающих (актуальных) по параметрам ?tabno=8337 или ?name=Кабанов (обрабатывается по принципу like) (облегчённые аттрибуты)
	http.HandleFunc("/get-act-employee-light", handlers.RestSendEmployeeLightVersionAttributes(ins))

	//------------------------------------------------------------------
	// birthday notifications
	http.HandleFunc("/bd-notifications/", handlers.RestSendBdNotifications(ins))

	// set oocouple birthday notifications по параметрам ?tabno=8337
	http.HandleFunc("/bd-oocouple/", handlers.RestSetOOCoupleForBdNotifications(ins))

	//------------------------------------------------------------------
	// server   --------------------------------------------------------
	//------------------------------------------------------------------
	addr := flag.String("addr", ":8080", "Сетевой адрес веб-сервера MD")
	flag.Parse()

	srv := &http.Server{
		Addr:    *addr,
		Handler: nil,
	}
	err := srv.ListenAndServe()
	log.Error("main: srv.ListenAndServe() error: %v", err)
}
