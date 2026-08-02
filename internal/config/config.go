package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	JWT      JWTConfig
	RabbitMQ RabbitMQConfig
}

type ServerConfig struct{ Port string }
type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
}
type RedisConfig struct {
	Host     string
	Port     string
	Password string
}
type JWTConfig struct {
	Secret      string
	ExpireHours int
}
type RabbitMQConfig struct {
	Host     string
	Port     string
	User     string
	Password string
}

func (c RabbitMQConfig) DSN() string {
	return fmt.Sprintf("amqp://%s:%s@%s:%s/", c.User, c.Password, c.Host, c.Port)
}

func Load() *Config {
	viper.SetConfigFile(".env")
	viper.AutomaticEnv()
	_ = viper.ReadInConfig()

	viper.SetDefault("SERVER_PORT", "8000")
	viper.SetDefault("JWT_EXPIRE_HOURS", 24)

	return &Config{
		Server: ServerConfig{Port: viper.GetString("SERVER_PORT")},
		Database: DatabaseConfig{
			Host:     viper.GetString("MYSQL_HOST"),
			Port:     viper.GetString("MYSQL_PORT"),
			User:     viper.GetString("MYSQL_USER"),
			Password: viper.GetString("MYSQL_PASSWORD"),
			DBName:   viper.GetString("MYSQL_DATABASE"),
		},
		Redis: RedisConfig{
			Host:     viper.GetString("REDIS_HOST"),
			Port:     viper.GetString("REDIS_PORT"),
			Password: viper.GetString("REDIS_PASSWORD"),
		},
		JWT: JWTConfig{
			Secret:      viper.GetString("JWT_SECRET"),
			ExpireHours: viper.GetInt("JWT_EXPIRE_HOURS"),
		},
		RabbitMQ: RabbitMQConfig{
			Host:     viper.GetString("RABBITMQ_HOST"),
			Port:     viper.GetString("RABBITMQ_PORT"),
			User:     viper.GetString("RABBITMQ_USER"),
			Password: viper.GetString("RABBITMQ_PASSWORD"),
		},
	}
}
