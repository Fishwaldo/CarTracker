package config

import (
	"fmt"

	"github.com/spf13/viper"
)

func init() {
	viper.SetConfigName("cartracker")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Errorf("can't read config file: %w", err))
	}
	fmt.Printf("Loaded Config File %s\n", viper.ConfigFileUsed())
	viper.SetDefault("name", "SMQ4629S")
}