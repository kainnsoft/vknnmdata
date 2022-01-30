package repository

import (
	"github.com/spf13/viper"
)

func ReadConfig(param string) string {
	configPath := "./config.yml"

	viper.AutomaticEnv()
	viper.SetConfigName("config")

	viper.SetConfigFile(configPath)
	viper.ReadInConfig()

	// если файл найден, вернем значение из файла, иначе default-ное
	return viper.GetString(param)

}
