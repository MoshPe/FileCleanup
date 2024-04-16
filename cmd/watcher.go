package cmd

import (
	log "github.com/sirupsen/logrus"
	"os"
)

func HandleNewFile(filePath string) {
	mutex.Lock()
	defer mutex.Unlock()

	fileInfo, err := os.Stat(filePath)
	if err != nil {
		log.Errorln("Error getting file info:", err)
		return
	}

	// Update fileInfoMap with the new file's information
	fileInfoMap[filePath] = FileInfo{
		Size:    fileInfo.Size(),
		ModTime: fileInfo.ModTime(),
	}

	if AppConfig.IsDetailedLogEnabled {
		log.Infoln("Added:", filePath)
	}
}
