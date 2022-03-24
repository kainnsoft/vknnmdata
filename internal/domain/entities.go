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
	RefApi1CZupPing            string
	RefApi1CZupAllEmployees    string
	RefPutTo1CZupEmailAdress   string
	RefApi1CZupAllDepartaments string
	RefPutTo1CCreateUserAdress string
}
