package cmd

import (
	constant "FileCleanup/const"
	"FileCleanup/pkg"
	log "github.com/sirupsen/logrus"
	"os"
	"sync"
	"time"
)

var (
	mutex sync.Mutex
)

func DeleteExcessFiles(config pkg.DeleteConfig) {
	mutex.Lock()
	defer mutex.Unlock()

	log.Println("Started processing deletion of excess files...")
	var folderSize int64
	var maxFolderSize int64
	var numFilesToDelete int64
	if config.MaxFolderPercentEnabled {
		var totalSize float64
		folderSize, _, totalSize = getFolderSizePercent(config)
		maxFolderSize = config.MaxFolderSizePercent
		numFilesToDelete = int64(totalSize) * (folderSize - maxFolderSize) / 100 / getFileAvgSizeMB()
		log.Printf("Folder size is %d%% out of allowed %d%%", folderSize, maxFolderSize)
	} else {
		folderSize = getFolderSizeMB()
		maxFolderSize = config.MaxFolderSizeMB
		log.Printf("Folder size is %f GB out of allowed %f GB", float64(folderSize)/constant.GB, float64(config.MaxFolderSizeMB)/constant.GB)
		numFilesToDelete = (folderSize - config.MaxFolderSizeMB) / getFileAvgSizeMB()
	}

	deletedFiles := uint64(0)

	if folderSize > maxFolderSize {
		log.Printf("Deleting excess files - %d files", numFilesToDelete)
		// Delete the oldest numFilesToDelete files
		for range numFilesToDelete {
			// Find the oldest file
			var oldestFile string
			var oldestTime time.Time
			for path, fileInfo := range fileInfoMap {
				if oldestTime.IsZero() || fileInfo.ModTime.Before(oldestTime) {
					oldestTime = fileInfo.ModTime
					oldestFile = path
				}
			}

			// Delete the oldest file
			if err := os.Remove(oldestFile); err != nil {
				log.Errorln("Error deleting file:", err)
				break // Exit loop on error
			}
			deletedFiles++
			if AppConfig.IsDetailedLogEnabled {
				log.Debugln("Deleted:", oldestFile)
			}

			// Update fileInfoMap after deletion
			delete(fileInfoMap, oldestFile)
		}
	}
	log.Println("Total deleted files", deletedFiles, "Remaining Folder size: ", getFolderSizeMB(), "MB")
}

func DeleteOldFiles(config pkg.DeleteConfig) {
	mutex.Lock()
	defer mutex.Unlock()

	log.Println("Started processing deletion of old files...")
	currentTime := time.Now()
	deletedFiles := uint64(0)
	if len(fileInfoMap) == 0 {
		log.Println("No files to delete")
		return
	}
	for path, fileInfo := range fileInfoMap {
		if currentTime.Sub(fileInfo.ModTime).Hours()/24 > config.RetentionDays {
			if err := os.Remove(path); err != nil {
				log.Println("Error deleting file:", err)
			} else {
				deletedFiles++
				if AppConfig.IsDetailedLogEnabled {
					log.Println("Deleted:", path)
				}
			}
			delete(fileInfoMap, path)
		}
	}
	log.Printf("Total deleted files %d | Remaining folder size %d MB", deletedFiles, getFolderSizeMB())
}
