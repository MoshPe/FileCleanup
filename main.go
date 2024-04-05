package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
)

const (
	targetFolder            = "/home/mosheper/GolandProjects/FileCleanup/junkFiles"
	retentionDays           = 30
	deleteIntervalSeconds   = 3600 // Change this to desired interval in seconds (e.g., 3600 for 1 hour)
	maxFolderSizeMB         = 256  // Maximum folder size in MB
	maxFolderSizePercent    = 70   // Maximum folder size as a percentage of the total drive size
	maxFolderPercentEnabled = true
	checkSizeIntervalSecs   = 600 // Interval for checking folder size in seconds (e.g., 600 for every 10 minutes)
	isDetailedLogEnabled    = false
)

const (
	KB = 1024
	MB = 1024 * KB
	GB = 1024 * MB
	TB = 1024 * GB
)

var (
	fileInfoMap  = make(map[string]FileInfo)
	driveInfo    = &syscall.Statfs_t{}
	driveInfoSet sync.Once
	mutex        sync.Mutex
)

type FileInfo struct {
	Size    int64
	ModTime time.Time
}

func main() {
	// Populate fileModTimes with existing files in the target folder
	if err := populateFileInfoMap(); err != nil {
		log.Fatal("Error populating fileModTimes:", err)
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
					if strings.HasPrefix(event.Name, targetFolder) {
						handleNewFile(event.Name)
					}
				}
			case err := <-watcher.Errors:
				if err != nil {
					log.Println("Watcher error:", err)
				}
			}
		}
	}()

	err = watcher.Add(targetFolder)
	if err != nil {
		log.Fatal(err)
	}

	// Start a ticker for periodic deletion
	ticker := time.NewTicker(time.Duration(deleteIntervalSeconds) * time.Second)
	defer ticker.Stop()

	// Start a ticker for monitoring folder size and triggering deletion
	sizeTicker := time.NewTicker(time.Duration(checkSizeIntervalSecs) * time.Second)
	defer sizeTicker.Stop()

	// Run the deletion process immediately and then periodically
	deleteOldFiles()
	for {
		select {
		case <-ticker.C:
			deleteOldFiles()
		case <-sizeTicker.C:
			go deleteExcessFiles()
		}
	}
	<-make(chan struct{})
}

func handleNewFile(filePath string) {
	mutex.Lock()
	defer mutex.Unlock()

	fileInfo, err := os.Stat(filePath)
	if err != nil {
		log.Println("Error getting file info:", err)
		return
	}

	// Update fileInfoMap with the new file's information
	fileInfoMap[filePath] = FileInfo{
		Size:    fileInfo.Size(),
		ModTime: fileInfo.ModTime(),
	}

	if isDetailedLogEnabled {
		log.Println("Added:", filePath)
	}
}

func populateFileInfoMap() error {
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

func deleteOldFiles() {
	mutex.Lock()
	defer mutex.Unlock()

	currentTime := time.Now()
	deletedFiles := uint64(0)
	if len(fileInfoMap) == 0 {
		log.Println("No files to delete")
		return
	}
	for path, fileInfo := range fileInfoMap {
		if currentTime.Sub(fileInfo.ModTime).Hours()/24 > retentionDays {
			if err := os.Remove(path); err != nil {
				log.Println("Error deleting file:", err)
			} else {
				deletedFiles++
				if isDetailedLogEnabled {
					log.Println("Deleted:", path)
				}
			}
			delete(fileInfoMap, path)
		}
	}
	log.Println("Total deleted files", deletedFiles)
}

func deleteExcessFiles() {
	mutex.Lock()
	defer mutex.Unlock()

	var folderSize int64
	var maxFolderSize int64
	var numFilesToDelete int64
	if maxFolderPercentEnabled {
		folderSize = getFolderSizePercent()
		maxFolderSize = maxFolderSizePercent
		numFilesToDelete = int64(float64(len(fileInfoMap)) * (float64(folderSize-maxFolderSizePercent) / float64(maxFolderSize)))
	} else {
		folderSize = getFolderSizeMB()
		maxFolderSize = maxFolderSizeMB
		// Calculate the number of files to delete
		numFilesToDelete = (folderSize - maxFolderSizeMB) / getFileAvgSizeMB()
	}

	deletedFiles := uint64(0)

	if folderSize > maxFolderSize {
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
				log.Println("Error deleting file:", err)
				break // Exit loop on error
			}
			deletedFiles++
			if isDetailedLogEnabled {
				log.Println("Deleted:", oldestFile)
			}

			// Update fileInfoMap after deletion
			delete(fileInfoMap, oldestFile)
		}
	}
	log.Println("Total deleted files", deletedFiles, "Remaining Folder size ", getFolderSizeMB(), "MB")
}

func getFolderSizeMB() int64 {
	var size int64

	for _, fileInfo := range fileInfoMap {
		size += fileInfo.Size
	}

	// Convert bytes to MB
	return size / (1024 * 1024)
}

func getFileAvgSizeMB() int64 {
	if len(fileInfoMap) == 0 {
		return 0
	}

	totalSize := int64(0)
	for _, fileInfo := range fileInfoMap {
		totalSize += fileInfo.Size
	}
	return totalSize / int64(len(fileInfoMap)) / (1024 * 1024)
}

func getFolderSizePercent() int64 {
	driveInfoSet.Do(func() {
		path := filepath.Dir(targetFolder)
		stat := syscall.Statfs_t{}
		if err := syscall.Statfs(path, &stat); err != nil {
			log.Fatalf("Failed to get file system information for %s: %v", path, err)
		}
		driveInfo = &stat
	})

	// Get the total size of the drive where the folder resides
	totalSize := driveInfo.Blocks * uint64(driveInfo.Bsize)

	// Calculate the total size of all files in the folder
	var folderSize int64
	for _, fileInfo := range fileInfoMap {
		folderSize += fileInfo.Size
	}

	// Calculate the folder size as a percentage of the total drive size
	folderSizePercent := int((float64(folderSize) / float64(totalSize)) * 100)
	log.Println("Folder size: ", folderSize/MB, "Total Drive size: ", totalSize/MB)

	return int64(folderSizePercent)
}
