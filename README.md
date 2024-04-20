# FileCleanup

## Introduction
FileCleanup CLI is a command-line tool designed to help you manage and maintain your file system by automating the cleanup of unnecessary files and folders. With its flexible configuration options, FileCleanup CLI allows you to define rules for deleting files based on criteria such as retention period, folder size limits, and interval-based deletion.

### Key Features:
- Flexible Configuration: Define cleanup rules using a JSON configuration file, allowing you to specify target folders, retention periods, delete intervals, maximum folder sizes, and more.
- Automatic Cleanup: Schedule automatic file cleanup tasks based on your defined rules and intervals, reducing manual intervention and ensuring your file system stays organized and efficient.
- Detailed Logging: Optionally enable detailed logging to track cleanup operations, providing visibility into deleted files, cleanup actions, and any errors encountered during the process.
- Customizable Policies: Tailor cleanup policies to suit your specific requirements, whether you need to enforce strict retention periods, limit folder sizes, or optimize disk space usage.
- Easy Integration: Seamlessly integrate FileCleanup CLI into your existing workflows or scheduled tasks, making it easy to incorporate automated file cleanup into your system maintenance routines.

## Getting Started
FileClieanCli generates its configueation upon its first running.<br>
Running `go run main.go` or `fileCleanup` will produce the configuration file at the following location:
- windows: C:/windows/$USER/.fileCleanup/fileCleanup.json
- linux: /home/$USER/.fileCleanup/fileCleanup.json

It is possible to alter the configuration path by utilizing the `-f` flag to access a custom configuration file.

## FileGenerator
This repo has a file generator to test the behaviour of the file cleanup. <br>Simple edit [FileGenerator](fileGenerator/main.go) with the desired # of files, file size, buffer and target folder. <br>
Simply run 
```go
go run fileGenerator/main.go
```

Example:
```go
go run fileGenerator/main.go
```
```text
Starting IO benchmark on Samsung SSD 990 Pro...
Disk type: Samsung SSD 990 Pro
Total operation Read, Write, Delete in seconds 0.3285541
Number of files: 10
File size: (KB) 10240
Total File size: (MB)  100
Average Write Speed (MB/s): 786.0251024976733
Total Write time in seconds: 0.1272224
Average Read Speed (MB/s): 2369.2191053828656
Total Read time in seconds: 0.042208
Average Delete Speed (MB/s): 15039.177056231481
Total Delete time in seconds: 0.0066493
Average Write IOPS: 201222.42623940436
Average Read IOPS: 606520.0909780136
Average Delete IOPS: 1503.9177056231483
Average CPU Time (s): 0.01272224
Average Delete CPU Time (s): 0.00066493
Average Memory Usage (bytes): 1.6961382113821135


Starting IO benchmark on Seagate Barracuda ST2000DM008...
Disk type: Seagate Barracuda ST2000DM008
Total operation Read, Write, Delete in seconds 7.9552990999999995
Number of files: 10
File size: (KB) 10240
Total File size: (MB)  100
Average Write Speed (MB/s): 12.88686414488691
Total Write time in seconds: 7.7598397
Average Read Speed (MB/s): 1403.4890738375602
Total Read time in seconds: 0.071251
Average Delete Speed (MB/s): 10403.012712481534
Total Delete time in seconds: 0.0096126
Average Write IOPS: 3299.037221091049
Average Read IOPS: 359293.2029024154
Average Delete IOPS: 1040.3012712481534
Average CPU Time (s): 0.77598397
Average Delete CPU Time (s): 0.0009612599999999999
Average Memory Usage (bytes): 1.335813492063492
```
