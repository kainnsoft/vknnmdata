package routes

import (
	"mdata/internal/handlers"
	"mdata/internal/repository"
	"net/http"
)

func InitializeRoutes(mux *http.ServeMux, ins *repository.PostgreInstance) {
	// ping - метод. Получим доступность базы 1С:ЗУП
	mux.HandleFunc("/from-zup/ping/", handlers.PingZup)

	// ping - метод. Получим всех пользователей 1С:ЗУП
	mux.HandleFunc("/from-zup/ping/AllUsers/", handlers.PingFromZmupAllUsers)

	// Запишем в БД всех user-ов из ЗУП-а, по пути получив их email из AD                     (daily task)
	mux.HandleFunc("/db/from-zup/write/all-users/", handlers.RestHandleZupWriteAllUsers(ins))

	// Debug-method. Запишем в БД одного user-а из запроса (из Postman-а), по пути получив его email из AD
	mux.HandleFunc("/db/from-zup/write/one-user/", handlers.RestHandleDebugWriteOneUser(ins))

	// Debug-method Отправляем в 1С:ЗУП email одного захардкоженного пользователя.
	mux.HandleFunc("/to-zup/send-email/ping/", handlers.RestPutOneUserEmailToZup)

	// Отправляем в 1С:ЗУП email всех пользователей для обновления email (можно один, но в массиве)    (daily task)
	mux.HandleFunc("/to-zup/send-email/put-array/", handlers.RestPutAllUsersWithEmailToZup(ins))

	// Отправляем в 1С:CreateUser даные новых сотрудников (users) для создания для них пользователей в 1С (можно один, но в массиве)    (daily task)
	mux.HandleFunc("/to-1cuser/send-accounts-crud/put-array/", handlers.RestPutNewUsersWithEmailToCreate1CAccounts(ins))

	// блок "Подразделения"
	// ping-метод - получить все подразделения из 1С:ЗУП
	mux.HandleFunc("/from-zup/ping/AllDepartaments/", handlers.RestGetFromZupPingAllDepartaments())

	// получить все подразделения из 1С:ЗУП и записать их в базу MD
	mux.HandleFunc("/db/from-zup/write/all-departaments/", handlers.RestHandleFromZupAllDepartament(ins))

	// Debug-method. Запишем в БД одно подразделение из запроса (из Postman-а)
	mux.HandleFunc("/db/from-zup/write/singl-departament/", handlers.RestHandleDebugWriteSingleDepartament(ins))

	//------------------------------------------------------------------
	// блок REST api  --------------------------------------------------
	//------------------------------------------------------------------
	// ping - метод. Доступность сервиса MD
	mux.HandleFunc("/", handlers.PingMasterData)

	// ping - метод. Доступность DB
	mux.Handle("/db/ping/", handlers.PingDB(ins))
	//------------------------------------------------------------------
	// все аттрибуты
	// отдать всех работающих (актуальных) физ.лиц (все аттрибуты)
	mux.HandleFunc("/get-act-employees/", handlers.RestSendActEmployeesAllAttributes(ins))

	// отдать всех работающих (актуальных) физ.лиц, у которых есть email-ы (все аттрибуты)
	mux.HandleFunc("/get-act-email-employees/", handlers.RestSendAllEmailEmployeesAllAttributes(ins))

	// отдать сотрудника(-ков) работающих (актуальных) по параметрам ?tabno=8337 или ?name=Кабанов (обрабатывается по принципу like) (все аттрибуты)
	mux.HandleFunc("/get-act-employee", handlers.RestSendEmployeeAllAttributes(ins))

	// отдать всех уволенных физ.лиц (все аттрибуты) по параметрам ?from=2006-01-02  (YYYY-MM-DD)
	mux.HandleFunc("/get-fired-employees", handlers.RestSendUsersFiredFromAllAttributes(ins))

	//------------------------------------------------------------------
	// облегчённые аттрибуты
	// отдать всех работающих (актуальных) физ.лиц (облегчённые аттрибуты)
	mux.HandleFunc("/get-act-employees-light/", handlers.RestSendAllEmployeesLightVersionAttributes(ins))

	// отдать всех работающих (актуальных) физ.лиц, у которых есть email-ы (облегчённые аттрибуты)
	mux.HandleFunc("/get-act-email-employees-light/", handlers.RestSendAllEmailEmployeesLightVersionAttributes(ins))

	// отдать сотрудника(-ков) работающих (актуальных) по параметрам ?tabno=8337 или ?name=Кабанов (обрабатывается по принципу like) (облегчённые аттрибуты)
	mux.HandleFunc("/get-act-employee-light", handlers.RestSendEmployeeLightVersionAttributes(ins))

	//------------------------------------------------------------------
	// birthday notifications
	mux.HandleFunc("/bd-notifications/", handlers.RestSendBdNotifications(ins))

	// set oocouple birthday notifications по параметрам ?tabno=8337
	mux.HandleFunc("/bd-oocouple/", handlers.RestSetOOCoupleForBdNotifications(ins))

}
