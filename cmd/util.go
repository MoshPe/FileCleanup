package cmd

import (
	constant "FileCleanup/const"
	"FileCleanup/pkg"
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"os"
	"runtime"
	"syscall"
	"unsafe"
)

func getFolderSizeMB() int64 {
	var size int64

	for _, fileInfo := range fileInfoMap {
		size += fileInfo.Size
	}

	// Convert bytes to MB
	return size / constant.MB
}

func getFileAvgSizeMB() int64 {
	if len(fileInfoMap) == 0 {
		return 0
	}

	totalSize := int64(0)
	for _, fileInfo := range fileInfoMap {
		totalSize += fileInfo.Size
	}
	return totalSize / int64(len(fileInfoMap)) / constant.MB
}

func getFolderSizePercent(config pkg.DeleteConfig) (int64, float64, float64) {
	var totalSize int64
	var freeSize int64
	switch goos := runtime.GOOS; goos {
	case "windows":
		kernelDLL := syscall.MustLoadDLL("kernel32.dll")
		GetDiskFreeSpaceExW := kernelDLL.MustFindProc("GetDiskFreeSpaceExW")

		var avail int64

		path := config.TargetFolder
		_, _, lastErr := GetDiskFreeSpaceExW.Call(
			uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(path))),
			uintptr(unsafe.Pointer(&freeSize)),
			uintptr(unsafe.Pointer(&totalSize)),
			uintptr(unsafe.Pointer(&avail)),
		)

		log.Debugln(lastErr)
		log.Debugf("Free: %f GB - Total: %f GB", float64(freeSize)/constant.GB, float64(totalSize)/constant.GB)
		break
	default:
		log.Fatal("Unsupported platform.")
	}

	// Calculate the total size of all files in the folder
	var folderSize int64
	for _, fileInfo := range fileInfoMap {
		folderSize += fileInfo.Size
	}

	log.Printf("Folder size: %f GB | Available size: %f GB | Total Drive size: %f GB", float64(folderSize)/constant.GB, float64(freeSize)/constant.GB, float64(totalSize)/constant.GB)

	// Calculate the folder size as a percentage of the total drive size
	var folderSizePercent int
	if config.MaxFolderPercentFromAvailableSize {
		folderSizePercent = int((float64(folderSize) / float64(freeSize)) * 100)
		return int64(folderSizePercent), float64(folderSize) / constant.MB, float64(freeSize) / constant.MB
	}
	folderSizePercent = int((float64(folderSize) / float64(totalSize)) * 100)
	return int64(folderSizePercent), float64(folderSize) / constant.MB, float64(totalSize) / constant.MB
}

func UnmarshalJson(filepath string, obj *pkg.Config) {
	fileBytes, err := os.ReadFile(filepath)
	if err != nil {
		log.Fatalf("Error in reading file - reading %s - %s", filepath, err)
	}
	err = json.Unmarshal(fileBytes, obj)
	if err != nil {
		log.Fatalf("Error in unmarshling json file - reading %s - %s", filepath, err)
	}
}
