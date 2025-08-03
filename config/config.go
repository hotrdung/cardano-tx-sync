// config/config.go
package config

import (
	"github.com/spf13/viper"
)

// Config holds the application configuration
type Config struct {
	Ogmios    OgmiosConfig    `mapstructure:"ogmios"`
	Kafka     KafkaConfig     `mapstructure:"kafka"`
	DB        PostgresConfig  `mapstructure:"db"`
	API       APIConfig       `mapstructure:"api"`
	ChainSync ChainSyncConfig `mapstructure:"chainsync"`
}

// OgmiosConfig holds the configuration for Ogmios
type OgmiosConfig struct {
	Endpoint string `mapstructure:"endpoint"`
}

// KafkaConfig holds the configuration for Kafka
type KafkaConfig struct {
	Brokers []string `mapstructure:"brokers"`
}

// PostgresConfig holds the configuration for the PostgreSQL database
type PostgresConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
	SSLMode  string `mapstructure:"sslmode"`
}

// APIConfig holds the configuration for the API server
type APIConfig struct {
	ListenAddress string `mapstructure:"listen_address"`
}

// ChainSyncConfig holds the configuration for the chainsync process
type ChainSyncConfig struct {
	MaxCheckpointsToKeep int `mapstructure:"max_checkpoints_to_keep"`
}

// LoadConfig reads configuration from file or environment variables.
func LoadConfig(path string) (config Config, err error) {
	viper.AddConfigPath(path)
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	viper.AutomaticEnv()

	err = viper.ReadInConfig()
	if err != nil {
		return
	}

	err = viper.Unmarshal(&config)
	return
}
