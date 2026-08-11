package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type Config struct {
	Server        ServerConfig
	Database      DatabaseConfig
	Redis         RedisConfig
	JWT           JWTConfig
	RabbitMQ      RabbitMQConfig
	Mail          MailConfig
	Elasticsearch ElasticsearchConfig
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

// ElasticsearchConfig ES 学习环境配置（7.17 单节点，无安全认证时 Username/Password 留空）
type ElasticsearchConfig struct {
	Host     string
	Port     string
	Username string
	Password string
}

func (c ElasticsearchConfig) Addr() string {
	return fmt.Sprintf("http://%s:%s", c.Host, c.Port)
}

// MailConfig SMTP 邮件配置；SMTPHost/TO 为空时邮件功能自动禁用
type MailConfig struct {
	SMTPHost     string
	SMTPPort     string
	SMTPUser     string
	SMTPPassword string
	SMTPFrom     string
	SMTPTo       string
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
		Mail: MailConfig{
			SMTPHost:     viper.GetString("SMTP_HOST"),
			SMTPPort:     viper.GetString("SMTP_PORT"),
			SMTPUser:     viper.GetString("SMTP_USER"),
			SMTPPassword: viper.GetString("SMTP_PASSWORD"),
			SMTPFrom:     viper.GetString("SMTP_FROM"),
			SMTPTo:       viper.GetString("SMTP_TO"),
		},
		Elasticsearch: ElasticsearchConfig{
			Host:     viper.GetString("ES_HOST"),
			Port:     viper.GetString("ES_PORT"),
			Username: viper.GetString("ES_USERNAME"),
			Password: viper.GetString("ES_PASSWORD"),
		},
	}
}
