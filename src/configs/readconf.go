package configs

import (
	"sync"

	"github.com/ilyakaznacheev/cleanenv"
	"go.uber.org/zap"
)

type Config struct {
	Database       Database
	Jwt            Jwt
	OauthGoogle    OauthGoogle
	OauthMicrosoft OauthMicrosoft
}

type Jwt struct {
	My_super_secret_key string `env:"MY_SUPER_SECRET_KEY"`
}

type Database struct {
	Dname    string `env:"DNAME"`
	Username string `env:"USERNAM"`
	Password string `env:"PASSWORD"`
	Host     string `env:"HOST"`
	Port     int    `env:"PORT"`
}

type OauthGoogle struct {
	ClientID     string `env:"GOOGLE_CLIENT_ID"`
	ClientSecret string `env:"GOOGLE_CLIENT_SECRET"`
}

type OauthMicrosoft struct {
	ClientID     string `env:"MICROSOFT_CLIENT_ID"`
	ClientSecret string `env:"MICROSOFT_CLIENT_SECRET"`
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
