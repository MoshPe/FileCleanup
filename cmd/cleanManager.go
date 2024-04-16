package cmd

import (
	"errors"
	"fmt"
	"github.com/fsnotify/fsnotify"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type FileInfo struct {
	Size    int64
	ModTime time.Time
}

var (
	fileInfoMap    map[string]FileInfo
	ConfigFilePath string
)

func cleanCmd() *cobra.Command {
	var compareCmd = &cobra.Command{
		Use:     "clean [...FLAGS]",
		Short:   "Clean files based on configuration file",
		Example: `fileCleanup --file / -f /path/to/config/file clean`,
		Run: func(cmd *cobra.Command, args []string) {
			fileInfoMap = make(map[string]FileInfo)

			if ConfigFilePath != "" {
				UnmarshalJson(ConfigFilePath, &AppConfig)
			}

			log.Infoln(`Initiating FileCleanup`)

			// Populate fileModTimes with existing files in the target folder
			for _, deleteConfig := range AppConfig.DeleteConfig {
				if err := populateFileInfoMap(deleteConfig.TargetFolder); err != nil {
					log.Fatal("Error populating fileModTimes:", err)
				}
			}

			// Start watching for runtime changes
			watcher, err := fsnotify.NewWatcher()
			if err != nil {
				panic(err)
			}
			defer func(watcher *fsnotify.Watcher) {
				err := watcher.Close()
				if err != nil {
					panic(err)
				}
			}(watcher)

			// Start the watcher goroutine
			go func() {
				for {
					select {
					case event := <-watcher.Events:
						if event.Op&fsnotify.Create == fsnotify.Create {
							if AppConfig.IsDetailedLogEnabled {
								log.Debugln("File creation notification :: File " + event.Name)
							}
							for _, deleteConfig := range AppConfig.DeleteConfig {
								if strings.HasPrefix(event.Name, deleteConfig.TargetFolder) {
									HandleNewFile(event.Name)
								}
							}

						}
					case err := <-watcher.Errors:
						if err != nil {
							log.Println("Watcher error:", err)
						}
					}
				}
			}()

			for _, deleteConfig := range AppConfig.DeleteConfig {
				err = watcher.Add(deleteConfig.TargetFolder)
				if err != nil {
					log.Fatal(err)
				}
			}

			deleteRetentionTickers := make([]*time.Ticker, len(AppConfig.DeleteConfig))
			// Start a ticker for periodic deletion
			for i, deleteConfig := range AppConfig.DeleteConfig {
				ticker := time.NewTicker(time.Duration(deleteConfig.DeleteIntervalSeconds) * time.Second)
				deleteRetentionTickers[i] = ticker
			}

			deleteExcessTickers := make([]*time.Ticker, len(AppConfig.DeleteConfig))
			// Start a ticker for monitoring folder size and triggering deletion
			for i, deleteConfig := range AppConfig.DeleteConfig {
				ticker := time.NewTicker(time.Duration(deleteConfig.CheckSizeIntervalSecs) * time.Second)
				deleteExcessTickers[i] = ticker
			}
			defer func() {
				for _, ticker := range deleteExcessTickers {
					ticker.Stop()
				}
				for _, ticker := range deleteRetentionTickers {
					ticker.Stop()
				}
			}()

			for {
				for i, ticker := range deleteRetentionTickers {
					select {
					case <-ticker.C:
						go DeleteOldFiles(AppConfig.DeleteConfig[i])
					}
				}
				for i, ticker := range deleteExcessTickers {
					select {
					case <-ticker.C:
						go DeleteExcessFiles(AppConfig.DeleteConfig[i])
					}
				}
			}
			<-make(chan struct{})
		},
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) > 2 {
				return errors.New("required at least 1 file to compare")
			}
			return nil
		},
	}
	compareFlags(compareCmd)
	return compareCmd
}

func init() {
	RootCmd.AddCommand(cleanCmd())
}
func compareFlags(cmd *cobra.Command) {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatalln(err)
		return
	}
	cmd.Flags().StringVarP(&ConfigFilePath, "file", "f", filepath.Join(home, ".fileCleanup", ".fileCleanup.json"), fmt.Sprintf("Config file path. default: %s", filepath.Join(home, ".fileCleanup", ".fileCleanup.json")))
}

func populateFileInfoMap(targetFolder string) error {
	if _, err := os.Stat(targetFolder); os.IsNotExist(err) {
		log.Warningln("Target folder does not exist:", targetFolder)
		return nil
	}
	if AppConfig.IsDetailedLogEnabled {
		log.Traceln("Populating files to watch - folder:", targetFolder)
	}
	err := filepath.Walk(targetFolder, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			fileInfoMap[path] = FileInfo{
				Size:    info.Size(),
				ModTime: info.ModTime(),
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}
