package configs

import (
	"sync"

	"github.com/ilyakaznacheev/cleanenv"
	"go.uber.org/zap"
)

type Config struct {
	Database Database
}

type Database struct {
	Dname    string `env:"DNAME"`
	Username string `env:"USERNAM"`
	Password string `env:"PASSWORD"`
	Host     string `env:"HOST"`
	Port     int    `env:"PORT"`
}

var (
	cfg  *Config
	once sync.Once
)

func LoadConfig() *Config {
	once.Do(func() {
		cfg = &Config{}

		err := cleanenv.ReadConfig("config.env", cfg)
		if err != nil {
			err = cleanenv.ReadEnv(cfg)
			if err != nil {
				zap.L().Error("Config error")
			}
		}
	})
	return cfg
}
