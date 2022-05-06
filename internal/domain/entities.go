package domain

//Структура конфига, которая включает в себя необходимые нам настройки соединения (сюда можно добавить любые другие поля для postgres типа ssl и т.д.)
type Config struct {
	DBHost       string
	DBPort       string
	DBUsername   string
	DBPassword   string
	DataBaseName string
	DBTimeout    int

	ADHost     string
	ADPort     string
	ADUsername string
	ADPassword string

	MailUser     string
	MailPassword string
	MailHost     string
	MailPort     string
	MailFrom     string

	// ext refs
	RefApi1CZupPing                  string
	RefApi1CZupAllEmployees          string
	RefPutTo1CZupEmailAdress         string
	RefApi1CZupAllDepartaments       string
	RefPutTo1CCreateUserAdress       string
	RefPutTo1CCreateUserAdress_login string
	RefPutTo1CCreateUserAdress_pass  string
}

// структура для записи ошибки (или "ok") обмена, когда отправляем данные для записи в 1С (в таблицу exchanges)
// UserGUID - guid user-а (поле rowdata таблицы)
// AttemptCount  -количество попыток обмена до текущего (сколько раз уже пробовали выгружать) (поле attempt_count таблицы)
type Exchange1СErrorsStruct struct {
	UserGUID     string `json:"userGuid"`
	AttemptCount int    `json:"attemptCount"`
}

// получаемая из 1С структура ответа (хардкордная, описание здесь -
// http://wiki.vodokanal-nn.ru/bin/view/ОПП%20-%20Инструкции%20для%20разработчиков/Не%201С/Master%20Data/Интеграции/MD%20--%3E%201С%3ACreateUser/Автоматическое%20создание%20пользователей/):
type Response1CUserStatusStruct struct {
	UserGuid string `json:"userGuid"`
	Status   string `json:"status"`
}

//--------------------------------------------
// тип для мапинга get-запросов по пользователям
type EmployeeFilterData struct {
	Name  string `mapper:"name" json:"name"`
	Tabno string `mapper:"tabno" json:"tabno"`
}

//--------------------------------------------
// Инициализация обмена:
type ExchangeStruct struct {
	ExchangeId uint   `json:"exID"`
	BaseID     uint   `json:"baseID"`
	ReasonID   uint   `json:"reasonID"`
	RowData    string `json:"rowData"`
}

//--------------------------------------------
// Шаблон дней недели для организации поиска ДР:
type NextWeekStruct struct {
	Monday    string
	Tuesday   string
	Wednesday string
	Thursday  string
	Friday    string
	Saturday  string
	Sunday    string
}

// Departaments types
type Departaments struct {
	Departaments []Departament `json:"departaments"`
}

type Departament struct {
	DepartamentGUID        string   `json:"departamentGuid"`
	DepartamentDescr       string   `json:"departamentDescr"`
	DepartamentParentGUID  string   `json:"departamentParentGuid"`
	DepartamentIdZUP       string   `json:"departamentId"`
	DepartamentParentIdZUP string   `json:"departamentParentId"`
	DepartamentNotUsedFrom CastDate `json:"dateClose"`
}
