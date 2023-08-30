package util

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	JWTSecret     string `mapstructure:"JWT_SECRET" validate:"required"`
	RedisAddress  string `mapstructure:"REDIS_ADDR" validate:"required"`
	RedisPassword string `mapstructure:"REDIS_PW"`
	Port          string `mapstructure:"PORT" validate:"required,number"`
}

// func LoadConfigViper(path string) (*Config, error) {
// 	var config *Config

// 	viper.AddConfigPath(path)
// 	viper.SetConfigName("app")
// 	viper.SetConfigType("env")
// 	viper.AutomaticEnv()

// 	err := viper.ReadInConfig()
// 	if err != nil {
// 		return nil, err
// 	}

// 	viper.BindEnv("JWT_SECRET")
// 	viper.BindEnv("REDIS_ADDR")
// 	viper.BindEnv("REDIS_PW")
// 	viper.BindEnv("PORT")

// 	err = viper.Unmarshal(&config)

// 	if err != nil {
// 		return nil, err
// 	}

// 	if err = Validate.Struct(config); err != nil {
// 		return nil, err
// 	}

// 	return config, nil
// }

func LoadConfig() (*Config, error) {
	godotenv.Load()

	config := &Config{
		JWTSecret:     os.Getenv("JWT_SECRET"),
		RedisAddress:  os.Getenv("REDIS_ADDR"),
		Port:          os.Getenv("PORT"),
		RedisPassword: os.Getenv("REDIS_PW"),
	}

	if err := Validate.Struct(config); err != nil {
		return nil, err
	}

	return config, nil
}
