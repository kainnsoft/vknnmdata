package repository

import (
	"mdata/internal/domain"
	"strconv"
)

var cfg *domain.Config = &domain.Config{} // TODO хранить в кэше с возможностью принудительного сброса

// ext refs
func init() {
	cfg.DBHost = ReadConfig("db.host")
	cfg.DBUsername = ReadConfig("db.username")
	cfg.DBPassword = ReadConfig("db.password")
	cfg.DBPort = ReadConfig("db.port")
	cfg.DataBaseName = ReadConfig("db.dbname")
	cfg.DBTimeout, _ = strconv.Atoi(ReadConfig("db.timeout"))

	cfg.ADHost = ReadConfig("ad.host")
	cfg.ADUsername = ReadConfig("ad.username")
	cfg.ADPassword = ReadConfig("ad.password")
	cfg.ADPort = ReadConfig("ad.port")

	cfg.MailUser = ReadConfig("mail.user")
	cfg.MailPassword = ReadConfig("mail.password")
	cfg.MailHost = ReadConfig("mail.host")
	cfg.MailPort = ReadConfig("mail.port")
	cfg.MailFrom = ReadConfig("mail.from")

	cfg.RefApi1CZupPing = ReadConfig("ref.api1czupping")
	cfg.RefApi1CZupAllEmployees = ReadConfig("ref.api1czupallemployees")
	cfg.RefPutTo1CZupEmailAdress = ReadConfig("ref.putto1czupemailadress")
	cfg.RefApi1CZupAllDepartaments = ReadConfig("ref.api1czupalldepartaments")
	cfg.RefPutTo1CCreateUserAdress = ReadConfig("ref.putto1ccreateuseradress")
}

func GetCfg() *domain.Config {
	return cfg
}

func GetApi1CZupPing() string {
	return cfg.RefApi1CZupPing
}

func GetApi1CZupAllEmployees() string {
	return cfg.RefApi1CZupAllEmployees
}

func GetPutTo1CZupEmailAdress() string {
	return cfg.RefPutTo1CZupEmailAdress
}

func GetApi1CZupAllDepartaments() string {
	return cfg.RefApi1CZupAllDepartaments
}

func GetPutTo1CCreateUserAdress() string {
	return cfg.RefPutTo1CCreateUserAdress
}

// название структуры, получаемой из 1С
const Array1Cname = "UsersStatus"

// статус записи - успешно ли произошла запись данных строки в 1С
const RowStatusSuccess = "Success"

// записываем в таблицу exchange, когда обмен прошёл успешно
const Response200ok = "200(ok)"
