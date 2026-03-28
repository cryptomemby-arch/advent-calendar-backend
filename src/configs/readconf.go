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
	Origin         Origin
	Photo          Photo
}

type Jwt struct {
	My_super_secret_key string `env:"MY_SUPER_SECRET_KEY"`
}

type Database struct {
	Dname    string `env:"DNAME"`
	Username string `env:"USERNAME"`
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

type Origin struct {
	OriginFront string `env:"ORIGINFRONT"`
	OriginBack  string `env:"ORIGINBACK"`
}

type Photo struct {
	CloudName      string `env:"CLOUD_NAME"`
	ApiKeyPhoto    string `env:"API_KEY_PHOTO"`
	ApiSecretPhoto string `env:"API_SECRET_PHOTO"`
	FolderPhoto    string `env:"FolderPhoto"`
}

var (
	cfg  *Config
	once sync.Once
)

func LoadConfig() *Config {
	once.Do(func() {
		cfg = &Config{}

		err := cleanenv.ReadConfig("src/configs/config.env", cfg)
		if err != nil {

			zap.L().Warn("Could not read config.env, trying environment variables", zap.Error(err))

			err = cleanenv.ReadEnv(cfg)
			if err != nil {
				zap.L().Fatal("Config error", zap.Error(err))
			}
		}
	})
	return cfg
}
