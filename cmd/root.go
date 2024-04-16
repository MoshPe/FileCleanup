package cmd

import (
	"FileCleanup/pkg"
	"encoding/json"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"log"
	"os"
	"path/filepath"
)

var cfgFile string
var AppConfig pkg.Config

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:     "FileCleanup",
	Short:   "A powerful tool for clean files based on their their relative size and modification date. ",
	Version: "1.0.0",
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	pkg.InitConfigDir()
	pkg.InitConfigFile()
	initConfig()
	InitLogger()

	RootCmd.CompletionOptions.DisableDefaultCmd = true
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		viper.AddConfigPath(filepath.Join(home, ".fileCleanup"))
		viper.SetConfigName(".fileCleanup")
		viper.SetConfigType("json")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		log.Fatal(err)
	}

	configMap := viper.AllSettings()

	_, err := json.Marshal(configMap)
	if err != nil {
		fmt.Println("Error marshaling config map:", err)
		return
	}

	// Unmarshal the JSON into Config struct
	//err = json.Unmarshal(jsonData, &AppConfig)
	//if err != nil {
	//	fmt.Println("Error unmarshalling config:", err)
	//	return
	//}
}
