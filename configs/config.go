package configs

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func InitConfig() error {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AutomaticEnv()
	viper.SetDefault("serpapi.api_key", "")

	err := viper.BindEnv("serpapi.api_key", "SERPAPI_KEY")
	if err != nil {
		log.Fatalf("error binding environment variable: %v", err)
	}

	if err := viper.ReadInConfig(); err != nil {
		// Handle the error if config file is not found
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Println("Config file not found; please set SERPAPI_KEY in environment or provide a config file.")
		} else {

			return fmt.Errorf("error reading config file: %w", err)
		}
	}

	return nil
}
