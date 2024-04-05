package main

import (
	"fmt"
	"os"
)

const (
	targetFolder = "/home/mosheper/GolandProjects/FileCleanup/junkFiles"
	fileSize     = 10 * 1024 * 1024 // 10MB file size
	numFiles     = 30000
)

func main() {
	if err := os.MkdirAll(targetFolder, 0755); err != nil {
		fmt.Println("Error creating target folder:", err)
		return
	}

	for i := 0; i < numFiles; i++ {
		fileName := fmt.Sprintf("%s/file%d.txt", targetFolder, i)
		if err := generateFile(fileName, fileSize); err != nil {
			fmt.Printf("Error generating file %s: %v\n", fileName, err)
		} else {
			fmt.Printf("Generated file %s\n", fileName)
		}
	}
}

func generateFile(fileName string, size int64) error {
	file, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write data to the file to reach the desired size
	data := make([]byte, size)
	if _, err := file.Write(data); err != nil {
		return err
	}

	return nil
}
