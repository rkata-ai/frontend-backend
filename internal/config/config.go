package config

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/viper"
)

type Config struct {
	Database DatabaseConfig `mapstructure:"database"`
}

type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
	SSLMode  string `mapstructure:"sslmode"`
}

func LoadConfig(configPath string) (*Config, error) {
	v := viper.New()

	if configPath != "" {
		dir, file := filepath.Split(configPath)
		ext := filepath.Ext(file)
		fileName := file[:len(file)-len(ext)]

		v.AddConfigPath(dir)
		v.SetConfigName(fileName)
		v.SetConfigType(ext[1:]) // remove the dot
	} else {
		v.AddConfigPath(".")
		v.SetConfigName("config") // default config.yaml
		v.SetConfigType("yaml")
	}

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unable to decode config into struct: %w", err)
	}

	return &cfg, nil
}
