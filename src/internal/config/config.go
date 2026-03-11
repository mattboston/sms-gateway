package config

import (
	"os"
	"strings"

	"github.com/spf13/viper"
)

// Config holds all application configuration.
type Config struct {
	DBDriver   string `mapstructure:"db_driver"`
	DBDSN      string `mapstructure:"db_dsn"`
	DevicePath string `mapstructure:"device_path"`
	BaudRate   int    `mapstructure:"baud_rate"`
	Port       int    `mapstructure:"port"`
	DevMode    bool   `mapstructure:"dev_mode"`
	JWTSecret  string `mapstructure:"jwt_secret"`
}

// Load reads configuration from the global viper instance (which has CLI flags bound)
// and environment variables, then returns a Config.
func Load() (*Config, error) {
	viper.SetDefault("db_driver", "sqlite")
	viper.SetDefault("db_dsn", "/opt/sms-gateway/sms-gateway.db")
	viper.SetDefault("device_path", "")
	viper.SetDefault("baud_rate", 9600)
	viper.SetDefault("port", 5174)
	viper.SetDefault("dev_mode", false)
	viper.SetDefault("jwt_secret", "change-me-in-production")
	viper.SetDefault("config_file", "/opt/sms-gateway/sms-gateway.conf")

	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	configFile := viper.GetString("config_file")
	if configFile != "" {
		if _, err := os.Stat(configFile); err == nil {
			viper.SetConfigFile(configFile)
			viper.SetConfigType("env")
			if err := viper.ReadInConfig(); err != nil {
				return nil, err
			}
		}
	}

	cfg := &Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
