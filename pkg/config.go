package pkg

import (
	"encoding/json"
	"errors"
	log "github.com/sirupsen/logrus"
	"os"
	"path/filepath"
)

type DeleteConfig struct {
	TargetFolder                      string  `json:"target_folder"`
	RetentionDays                     float64 `json:"retention_days"`
	DeleteIntervalSeconds             int     `json:"delete_interval_seconds"`
	MaxFolderSizeMB                   int64   `json:"max_folder_size_mb"`
	MaxFolderSizePercent              int64   `json:"max_folder_size_percent"`
	MaxFolderPercentEnabled           bool    `json:"max_folder_percent_enabled"`
	MaxFolderPercentFromAvailableSize bool    `json:"max_folder_percent_from_available_size"`
	CheckSizeIntervalSecs             int     `json:"check_size_interval_secs"`
}

type Config struct {
	DeleteConfig         []DeleteConfig `json:"delete_config"`
	IsDetailedLogEnabled bool           `json:"detailed_log"`
	LogFilePath          string         `json:"log_file_path"`
}

func InitConfigDir() {
	dirname, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dirname, ".fileCleanup")); errors.Is(err, os.ErrNotExist) {
		err := os.Mkdir(filepath.Join(dirname, ".fileCleanup"), 0777)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func InitConfigFile() {
	dirname, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dirname, ".fileCleanup", ".fileCleanup.json")); errors.Is(err, os.ErrNotExist) {
		file, err := os.Create(filepath.Join(dirname, ".fileCleanup", ".fileCleanup.json"))
		if err != nil {
			log.Fatal(err)
		}
		log.Infof("Created configuration file %s at %s", file.Name(), filepath.Join(dirname, ".fileCleanup"))
		initJsonData(file)
	}
}

func initJsonData(file *os.File) {
	result := Config{}
	result.LogFilePath = "/path/to/reference/folder"
	result.IsDetailedLogEnabled = false
	result.DeleteConfig = []DeleteConfig{
		{
			TargetFolder:                      result.LogFilePath,
			RetentionDays:                     5,
			DeleteIntervalSeconds:             24 * 60 * 60,
			MaxFolderSizeMB:                   1000,
			MaxFolderSizePercent:              0,
			MaxFolderPercentEnabled:           false,
			MaxFolderPercentFromAvailableSize: false,
			CheckSizeIntervalSecs:             12 * 60 * 60,
		},
	}
	byteValue, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		log.Fatalln(err)
	}

	// Write back to file

	_, err = file.Write(byteValue)
	if err != nil {
		log.Fatal(err)
	}
}
