package config

import (
	"fmt"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/spf13/viper"
)

// depricated
func ReadConfig(param string) string {
	configPath := "./.env" //"./config.yml"

	viper.AutomaticEnv()
	viper.SetConfigName("config")

	viper.SetConfigFile(configPath)
	viper.ReadInConfig()

	// если файл найден, вернем значение из файла, иначе default-ное
	return viper.GetString(param)

}

const configPath string = "./.env"

type (
	// Config -.
	Config struct {
		App   `yaml:"app"`
		HTTP  `yaml:"http"`
		Log   `yaml:"logger"`
		PG    `yaml:"pgdb"`
		GRPC  `yaml:"grpc"`
		Kafka `yaml:"kafka"`
	}

	// App -.
	App struct {
		Name    string `yaml:"name"    env:"APP_NAME"`    // env-required:"true"
		Version string `yaml:"version" env:"APP_VERSION"` // env-required:"true"
	}

	// HTTP -.
	HTTP struct {
		HTTPAddr         string `yaml:"addr" env:"HTTP_ADDR" env-default:"localhost:8080"`
		HTTPReadTimeout  int    `env-required:"true" yaml:"read_timeout"  env:"HTTP_READTIMEOUT"`
		HTTPWriteTimeout int    `env-required:"true" yaml:"write_timeout" env:"HTTP_WRITETIMEOUT"`
	}

	// Log -.
	Log struct {
		InfoPath  string `yaml:"log_info_path"  env:"LOG_INFO_PATH"  env-default:"osStdOut"`
		ErrorPath string `yaml:"log_error_path" env:"LOG_ERROR_PATH" env-default:"osStdErr"`
		BuhPath   string `yaml:"log_buh_path"   env:"LOG_BUH_PATH" env-default:"osStdOut"`
	}

	// PG -.
	PG struct {
		Host        string `env-required:"true" yaml:"host"         env:"PG_HOST"`
		Username    string `env-required:"true" yaml:"username"     env:"PG_USERNAME"`
		Password    string `env-required:"true" yaml:"password"     env:"PG_PASSWORD"`
		Port        int    `env-required:"true" yaml:"port"         env:"PG_PORT"         env-default:"5432"`
		DBName      string `env-required:"true" yaml:"dbname"       env:"PG_DBNAME"`
		ConnTimeout int    `env-required:"true" yaml:"conn_timeout" env:"PG_CONN_TIMEOUT"`
		PoolMax     int    `env-required:"true" yaml:"pool_max"     env:"PG_POOL_MAX"     env-default:"5"`
	}

	// grpc
	GRPC struct {
		GRPCAddress string `yaml:"grpc_addr" env:"GRPC_ADDR" env-default:":50051"`
	}

	// kafka
	Kafka struct {
		KafkaAddress   string `yaml:"kafka_addr" env:"KAFKA_ADDR" env-default:":9092"`
		KafkaTopicMail string `yaml:"kafka_topic_mail" env:"KAFKA_TOPIC_MAIL" env-default:"md-topic-mail"`
	}
)

// NewConfig returns app config.
func NewConfig() (*Config, error) {
	cfg := &Config{}

	err := cleanenv.ReadConfig(configPath, cfg)
	if err != nil {
		return nil, fmt.Errorf("config error: %w", err)
	}

	return cfg, nil
}

func GetHTTPAddr() (string, error) {
	cfg := &Config{}

	err := cleanenv.ReadConfig(configPath, cfg)
	if err != nil {
		return "", fmt.Errorf("config error: %w", err)
	}

	return cfg.HTTPAddr, nil
}
