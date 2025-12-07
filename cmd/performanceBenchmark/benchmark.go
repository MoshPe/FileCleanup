package benchmark

import (
	"FileCleanup/cmd"
	_const "FileCleanup/const"
	"errors"
	"fmt"
	"image/color"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type Result struct {
	disk           string
	targetFolder   string
	color          color.Color
	writeTimes     []float64
	readTimes      []float64
	writeSpeeds    []float64
	readSpeeds     []float64
	writeIOPSs     []float64
	readIOPSs      []float64
	cpuTimes       []float64
	memUsages      []float64
	deleteSpeeds   []float64
	cpuDeleteTimes []float64
	deleteTimes    []float64
}

var (
	TestFilesPath  string
	FileSizeMB     int64
	BufferSizeByte int64
	NumberOfFiles  int64
	Name           string
	Parallel       bool
	Mode           string
)

var (
	startWrite  time.Time
	startRead   time.Time
	startDelete time.Time
)

func benchmarkCmd() *cobra.Command {
	var benchCmd = &cobra.Command{
		Use:     "benchmark [...FLAGS]",
		Short:   "Benchmark the IO performance of the disk",
		Example: `fileCleanup benchmark`,
		Run: func(cmd *cobra.Command, args []string) {

			result := Result{
				disk:         Name,
				targetFolder: TestFilesPath,
				color:        randomColor(),
			}

			var (
				totalWriteSpeed    float64
				totalReadSpeed     float64
				totalWriteIOPS     float64
				totalReadIOPS      float64
				totalCPUTime       float64
				totalMemUsage      float64
				totalDeleteCPUTime float64
				totalDeleteSpeed   float64
				mutex              sync.Mutex // Required for Parallel safety
				wg                 sync.WaitGroup
			)
			// Create target folder if it doesn't exist
			if err := os.MkdirAll(result.targetFolder, os.ModePerm); err != nil {
				log.Errorln("Error creating target folder:", err)
				return
			}

			totalWriteReadOps := (FileSizeMB * NumberOfFiles) / BufferSizeByte
			actualSizeMB := float64(FileSizeMB) / float64(_const.MB)

			log.Println("Disk type:", result.disk)
			log.Println("Parallel Mode:", Parallel) // Helpful log
			log.Println("Buffer size in KB:", BufferSizeByte/_const.KB)
			log.Println("Number of files:", NumberOfFiles)
			log.Println("File size (KB):", FileSizeMB/_const.KB)
			log.Println("Total File size (MB):", float64(FileSizeMB*NumberOfFiles)/_const.MB)
			log.Println("Total File size (GB):", float64(FileSizeMB*NumberOfFiles)/_const.GB)

			// Define file paths for benchmarking
			filePaths := make([]string, 0, NumberOfFiles)
			for i := 0; i < int(NumberOfFiles); i++ {
				filePath := filepath.Join(result.targetFolder, fmt.Sprintf("file_%d.dat", i))
				filePaths = append(filePaths, filePath)
			}

			log.Infoln("Starting IO benchmark on " + result.disk + "...")
			log.Infof("Mode: %s | Parallel: %v\n", Mode, Parallel)

			// Start CPU profile
			cpuProfile, err := os.Create("cpu_" + result.disk + "_profile.prof")
			if err != nil {
				log.Errorln("Error creating CPU profile:", err)
				return
			}

			if err := pprof.StartCPUProfile(cpuProfile); err != nil {
				log.Errorln("Error starting CPU profile:", err)
				return
			}
			totalTime := time.Now()
			if Mode == "all" || Mode == "write" {
				startWrite = time.Now()
				log.Infoln("--- Starting Write Test ---")
				// Collect benchmark results
				for _, path := range filePaths {
					worker := func(p string) {
						writeSpeed, writeIOPS, cpuTime, writeTime, err := measureWriteBenchmark(path)
						if err != nil {
							log.Errorf("Error writing file on %s: %v\n", path, err)
							return
						}
						mutex.Lock()
						result.writeSpeeds = append(result.writeSpeeds, writeSpeed)
						totalWriteSpeed += writeSpeed
						result.writeIOPSs = append(result.writeIOPSs, writeIOPS)
						totalWriteIOPS += writeIOPS
						result.cpuTimes = append(result.cpuTimes, cpuTime)
						totalCPUTime += cpuTime
						result.writeTimes = append(result.writeTimes, writeTime)
						mutex.Unlock()
					}

					if Parallel {
						wg.Add(1)
						go func(p string) {
							defer wg.Done()
							worker(p)
						}(path)
					} else {
						worker(path)
					}
				}
				if Parallel {
					wg.Wait()
				}
				stopWriteTime := time.Now()
				writeDuration := stopWriteTime.Sub(startWrite)
				actualSizeMB := float64(FileSizeMB) / float64(_const.MB)
				totalDataMB := float64(NumberOfFiles) * actualSizeMB
				avgWriteSpeed := totalDataMB / writeDuration.Seconds()
				avgWriteIOPS := float64(totalWriteReadOps) / writeDuration.Seconds()
				log.Println("Average Write Speed (MB/s):", avgWriteSpeed)
				log.Println("Total Write time in seconds:", writeDuration.Seconds())
				log.Println("Average Write IOPS:", avgWriteIOPS)
			}

			if Mode == "all" || Mode == "read" {
				startRead = time.Now()
				for _, path := range filePaths {
					worker := func(p string) {
						readSpeed, readIOPS, _, memUsage, readTime, err := measureReadBenchmark(path)
						if err != nil {
							log.Errorf("Error reading file on %s: %v\n", path, err)
							return
						}

						result.readSpeeds = append(result.readSpeeds, readSpeed)
						totalReadSpeed += readSpeed
						result.readIOPSs = append(result.readIOPSs, readIOPS)
						totalReadIOPS += readIOPS
						result.memUsages = append(result.memUsages, memUsage)
						totalMemUsage += memUsage
						result.readTimes = append(result.readTimes, readTime)
					}

					if Parallel {
						wg.Add(1)
						go func(p string) {
							defer wg.Done()
							worker(p)
						}(path)
					} else {
						worker(path)
					}
				}

				if Parallel {
					wg.Wait()
				}

				stopReadTime := time.Now()
				readDuration := stopReadTime.Sub(startRead)
				avgReadSpeed := (float64(NumberOfFiles) * actualSizeMB) / readDuration.Seconds()
				avgReadIOPS := float64(totalWriteReadOps) / readDuration.Seconds()
				log.Println("Average Read IOPS:", avgReadIOPS)
				log.Println("Average Read Speed (MB/s):", avgReadSpeed)
				log.Println("Total Read time in seconds:", readDuration.Seconds())
			}

			if Mode == "all" || Mode == "delete" {
				startDelete = time.Now()
				for _, path := range filePaths {
					worker := func(p string) {
						deleteSpeed, cpuTime, deleteTime, err := measureDeleteBenchmark(path)
						if err != nil {
							log.Errorf("Error reading file on %s: %v\n", path, err)
							return
						}

						result.deleteSpeeds = append(result.deleteSpeeds, deleteSpeed)
						totalDeleteSpeed += deleteSpeed
						result.cpuDeleteTimes = append(result.cpuDeleteTimes, cpuTime)
						totalDeleteCPUTime += cpuTime
						result.deleteTimes = append(result.deleteTimes, deleteTime)
					}

					if Parallel {
						wg.Add(1)
						go func(p string) {
							defer wg.Done()
							worker(p)
						}(path)
					} else {
						worker(path)
					}
				}

				if Parallel {
					wg.Wait()
				}

				stopDeleteTime := time.Now()
				deleteDuration := stopDeleteTime.Sub(startDelete)
				avgDeleteSpeed := (float64(NumberOfFiles) * actualSizeMB) / deleteDuration.Seconds()
				avgDeleteIOPS := float64(NumberOfFiles) / deleteDuration.Seconds()
				avgDeleteCPUTime := totalDeleteCPUTime / float64(NumberOfFiles)

				log.Println("Average Delete Speed (MB/s):", avgDeleteSpeed)
				log.Println("Total Delete time in seconds:", deleteDuration.Seconds())
				log.Println("Average Delete IOPS:", avgDeleteIOPS)
				log.Println("Average Delete CPU Time (s):", avgDeleteCPUTime)
			}

			cpuProfile.Close()
			pprof.StopCPUProfile()

			// Calculate averages
			avgCPUTime := totalCPUTime / float64(NumberOfFiles)
			avgMemUsage := totalMemUsage / float64(NumberOfFiles)

			// Print statistics
			log.Println("Average CPU Time (s):", avgCPUTime)
			log.Println("Average Memory Usage (bytes):", avgMemUsage)
			log.Println("Total Benchmark Time:", time.Since(totalTime).Seconds())
		},
		Args: func(cmd *cobra.Command, args []string) error {
			InitFilesPath()
			FileSizeMB = FileSizeMB * _const.MB
			return nil
		},
	}
	compareFlags(benchCmd)
	return benchCmd
}

func init() {
	cmd.RootCmd.AddCommand(benchmarkCmd())
}

func compareFlags(cmd *cobra.Command) {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatalln(err)
		return
	}
	cmd.Flags().StringVarP(&TestFilesPath, "path", "p", filepath.Join(home, ".fileCleanup", "benchmarkFiles"), fmt.Sprintf("Test files path. default: %s", filepath.Join(home, ".fileCleanup", "benchmarkFiles")))
	cmd.Flags().Int64VarP(&FileSizeMB, "size", "s", 10, "File size in MB")
	// 4 * 1024 * 1024 = 4194304 bytes (4MB)
	cmd.Flags().Int64VarP(&BufferSizeByte, "buffer", "b", 4194304, "Buffer size in bytes")
	cmd.Flags().Int64VarP(&NumberOfFiles, "count", "c", 1000, "Number of files")
	cmd.Flags().StringVarP(&Name, "name", "n", "", "Disk Name")
	cmd.Flags().BoolVarP(&Parallel, "parallel", "m", false, "Run in parallel (true) or sequential (false)")
	cmd.Flags().StringVarP(&Mode, "mode", "M", "all", "Benchmark mode: 'all', 'write', 'read', 'delete'")
}

func measureDeleteBenchmark(filePath string) (float64, float64, float64, error) {
	start := time.Now()
	startCPU := time.Now()
	var deleteSpeed float64

	// Perform file deletion
	if err := deleteBenchmark(filePath); err != nil {
		return 0, 0, 0, err
	}

	cpuTime := time.Since(startCPU).Seconds()
	deleteTime := time.Since(start).Seconds()

	// Calculate delete speed (MB/s)
	if deleteTime != 0 {
		deleteSpeed = float64(FileSizeMB) / deleteTime / (1024 * 1024)
	}

	return deleteSpeed, cpuTime, time.Since(startDelete).Seconds(), nil
}

func deleteBenchmark(filePath string) error {
	// Perform file deletion
	if err := os.Remove(filePath); err != nil {
		return err
	}
	return nil
}

// measureWriteBenchmark measures write speed, IOPS, and CPU time for a single file write operation
func measureWriteBenchmark(filePath string) (float64, float64, float64, float64, error) {
	startCPU := time.Now()

	// Write data to the file
	duration, err := writeBenchmark(filePath)
	if err != nil {
		return 0, 0, 0, 0, err
	}

	cpuTime := time.Since(startCPU).Seconds()

	// Calculate write speed (MB/s)
	writeSpeed := float64(FileSizeMB) / duration / (1024 * 1024)

	// Calculate write IOPS
	writeIOPS := float64(FileSizeMB) / float64(BufferSizeByte) / duration

	return writeSpeed, writeIOPS, cpuTime, time.Since(startWrite).Seconds(), nil
}

// measureReadBenchmark measures read speed, IOPS, CPU time, and memory usage for a single file read operation
func measureReadBenchmark(filePath string) (float64, float64, float64, float64, float64, error) {
	start := time.Now()
	startCPU := time.Now()

	// Read data from the file
	bytesRead, err := readBenchmark(filePath)
	if err != nil {
		return 0, 0, 0, 0, 0, err
	}

	duration := time.Since(start)
	cpuTime := time.Since(startCPU).Seconds()

	// Calculate read speed (MB/s)
	readSpeed := float64(bytesRead) / duration.Seconds() / (1024 * 1024)

	// Calculate read IOPS
	readIOPS := float64(bytesRead) / float64(BufferSizeByte) / duration.Seconds()

	// Calculate memory usage
	memUsage := getCpuInfo()

	return readSpeed, readIOPS, cpuTime, memUsage, time.Since(startRead).Seconds(), nil
}

// writeBenchmark writes a file with random data to the specified path
func writeBenchmark(filePath string) (float64, error) {
	file, err := os.Create(filePath)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	// 1. Create the data pattern (A, B, C...)
	buf := make([]byte, BufferSizeByte)
	for i := range buf {
		buf[i] = byte(i % 255)
	}

	start := time.Now()

	// 2. Write DIRECTLY to file (No bufio)
	remainingBytes := FileSizeMB // Remember: FileSizeMB is actually total bytes in your code
	for remainingBytes > 0 {
		// If remaining bytes is smaller than buffer, slice the buffer
		toWrite := buf
		if int64(len(buf)) > remainingBytes {
			toWrite = buf[:remainingBytes]
		}

		nn, err := file.Write(toWrite)
		if err != nil {
			return 0, err
		}
		remainingBytes -= int64(nn)
	}

	// 3. Force the OS to flush to disk (Critical for accurate benchmarking)
	err = file.Sync()
	if err != nil {
		return 0, err
	}

	since := time.Since(start)
	return since.Seconds(), nil
}

// readBenchmark reads the file from the specified path
func readBenchmark(filePath string) (int64, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	// Create a buffer to read data from the file
	buffer := make([]byte, BufferSizeByte)

	// Read data from the file
	var totalBytesRead int64
	for {
		bytesRead, err := file.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, err
		}
		totalBytesRead += int64(bytesRead)
	}
	return totalBytesRead, nil
}

// getCpuInfo retrieves CPU information about the current process
func getCpuInfo() float64 {
	var stat runtime.MemStats
	runtime.ReadMemStats(&stat)
	return float64(stat.Sys) / float64(stat.HeapSys)
}

// plotMetrics creates and saves separate plots for each collected metric with axis labels

func randomColor() color.RGBA {
	randGen := rand.New(rand.NewSource(time.Now().UnixNano()))
	return color.RGBA{R: uint8(randGen.Intn(256)), G: uint8(randGen.Intn(256)), B: uint8(randGen.Intn(256)), A: 255}
}

func InitFilesPath() {
	if _, err := os.Stat(TestFilesPath); errors.Is(err, os.ErrNotExist) {
		err := os.Mkdir(TestFilesPath, 0777)
		if err != nil {
			log.Fatal(err)
		}
	}
}
