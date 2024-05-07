package benchmark

import (
	"FileCleanup/cmd"
	_const "FileCleanup/const"
	"bufio"
	"errors"
	"fmt"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"gonum.org/v1/plot/vg"
	"image/color"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"time"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
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
			if NumberOfFiles == 1 {
				err := WriteFile(FileSizeMB, TestFilesPath)
				if err != nil {
					log.Fatalln(os.Stderr, FileSizeMB, err)
				}
				return
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
			)
			// Create target folder if it doesn't exist
			if err := os.MkdirAll(result.targetFolder, os.ModePerm); err != nil {
				log.Errorln("Error creating target folder:", err)
				return
			}

			// Define file paths for benchmarking
			filePaths := make([]string, 0, NumberOfFiles)
			for i := 0; i < int(NumberOfFiles); i++ {
				filePath := filepath.Join(result.targetFolder, fmt.Sprintf("file%s.dat", uuid.New()))
				filePaths = append(filePaths, filePath)
			}

			log.Infoln("Starting IO benchmark on " + result.disk + "...")

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
			startWrite = time.Now()
			// Collect benchmark results
			for _, path := range filePaths {
				writeSpeed, writeIOPS, cpuTime, writeTime, err := measureWriteBenchmark(path)
				if err != nil {
					log.Errorf("Error writing file on %s: %v\n", path, err)
					return
				}

				result.writeSpeeds = append(result.writeSpeeds, writeSpeed)
				totalWriteSpeed += writeSpeed
				result.writeIOPSs = append(result.writeIOPSs, writeIOPS)
				totalWriteIOPS += writeIOPS
				result.cpuTimes = append(result.cpuTimes, cpuTime)
				totalCPUTime += cpuTime
				result.writeTimes = append(result.writeTimes, writeTime)
			}
			stopWriteTime := time.Now()
			writeDuration := stopWriteTime.Sub(startWrite)

			startRead = time.Now()
			for _, path := range filePaths {
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

			stopReadTime := time.Now()
			readDuration := stopReadTime.Sub(startRead)

			startDelete = time.Now()
			for _, path := range filePaths {
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

			stopDeleteTime := time.Now()
			deleteDuration := stopDeleteTime.Sub(startDelete)
			cpuProfile.Close()
			pprof.StopCPUProfile()

			// Calculate averages
			avgWriteSpeed := float64(NumberOfFiles) * 10 / writeDuration.Seconds()
			avgReadSpeed := float64(NumberOfFiles) * 10 / readDuration.Seconds()
			totalWriteReadOps := (FileSizeMB * NumberOfFiles) / BufferSizeByte
			avgWriteIOPS := float64(totalWriteReadOps) / writeDuration.Seconds()
			avgReadIOPS := float64(totalWriteReadOps) / readDuration.Seconds()
			avgCPUTime := totalCPUTime / float64(NumberOfFiles)
			avgMemUsage := totalMemUsage / float64(NumberOfFiles)
			avgDeleteSpeed := float64(NumberOfFiles) * 10 / deleteDuration.Seconds()
			avgDeleteIOPS := float64(NumberOfFiles) / deleteDuration.Seconds()
			avgDeleteCPUTime := totalDeleteCPUTime / float64(NumberOfFiles)

			// Print statistics
			log.Println("Disk type:", result.disk)
			log.Println("Total operation Read, Write, Delete in seconds", time.Since(totalTime).Seconds())
			log.Println("Number of files:", NumberOfFiles)
			log.Println("File size (KB):", FileSizeMB/_const.KB)
			log.Println("Total File size (MB):", float64(FileSizeMB*NumberOfFiles)/_const.MB)
			log.Println("Total File size (GB):", float64(FileSizeMB*NumberOfFiles)/_const.GB)
			log.Println("Average Write Speed (MB/s):", avgWriteSpeed)
			log.Println("Total Write time in seconds:", writeDuration.Seconds())
			log.Println("Average Read Speed (MB/s):", avgReadSpeed)
			log.Println("Total Read time in seconds:", readDuration.Seconds())
			log.Println("Average Delete Speed (MB/s):", avgDeleteSpeed)
			log.Println("Total Delete time in seconds:", deleteDuration.Seconds())
			log.Println("Average Write IOPS:", avgWriteIOPS)
			log.Println("Average Read IOPS:", avgReadIOPS)
			log.Println("Average Delete IOPS:", avgDeleteIOPS)
			log.Println("Average CPU Time (s):", avgCPUTime)
			log.Println("Average Delete CPU Time (s):", avgDeleteCPUTime)
			log.Println("Average Memory Usage (bytes):", avgMemUsage)
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
	cmd.Flags().Int64VarP(&BufferSizeByte, "buffer", "b", 4096, "Buffer size in bytes")
	cmd.Flags().Int64VarP(&NumberOfFiles, "count", "c", 1000, "Number of files")
	cmd.Flags().StringVarP(&Name, "name", "n", "", "Disk Name")
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

	buf := make([]byte, BufferSizeByte)
	buf[len(buf)-1] = '\n'
	w := bufio.NewWriterSize(file, len(buf))

	start := time.Now()
	written := int64(0)
	for i := int64(0); i < FileSizeMB; i += int64(len(buf)) {
		nn, err := w.Write(buf)
		written += int64(nn)
		if err != nil {
			return 0, err
		}
	}
	err = w.Flush()
	if err != nil {
		return 0, err
	}
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
func plotMetrics(results []Result) error {
	// Create a separate plot for each metric
	plots := make([]*plot.Plot, 9)

	// Define the metrics names Write -> Read -> Delete
	metricNames := []string{"Write Speed MB", "Write IOPS", "Write - CPU Time", "Read Speed MB", "Read IOPS", "Memory Usage", "Delete Speed MB", "Delete - CPU Time"}

	// Create plots
	for i, name := range metricNames {
		plots[i] = plot.New()
		plots[i].X.Label.Text = "Seconds"
		plots[i].Y.Label.Text = name
	}

	for _, result := range results {
		// Add data to plots
		dataWrite := [][]float64{result.writeSpeeds, result.writeIOPSs, result.cpuTimes}
		dataRead := [][]float64{result.readSpeeds, result.readIOPSs, result.memUsages}
		dataDelete := [][]float64{result.deleteSpeeds, result.cpuDeleteTimes}
		i := 0
		for _, plotData := range dataWrite {
			points := make(plotter.XYs, len(plotData))
			for j := range points {
				points[j].X = result.readTimes[j]
				points[j].Y = plotData[j]
			}
			line, err := plotter.NewLine(points)
			if err != nil {
				return err
			}
			line.Color = result.color
			plots[i].Add(line)

			// Add legend and axis labels
			plots[i].Legend.Add(result.disk, line)

			i++
		}

		for _, plotData := range dataRead {
			points := make(plotter.XYs, len(plotData))
			for j := range points {
				points[j].X = result.writeTimes[j]
				points[j].Y = plotData[j]
			}
			line, err := plotter.NewLine(points)
			if err != nil {
				return err
			}
			line.Color = result.color
			plots[i].Add(line)

			// Add legend and axis labels
			plots[i].Legend.Add(result.disk, line)

			i++
		}

		for _, plotData := range dataDelete {
			points := make(plotter.XYs, len(plotData))
			for j := range points {
				points[j].X = result.deleteTimes[j]
				points[j].Y = plotData[j]
			}
			line, err := plotter.NewLine(points)
			if err != nil {
				return err
			}
			line.Color = result.color
			plots[i].Add(line)

			// Add legend and axis labels
			plots[i].Legend.Add(result.disk, line)

			i++
		}

	}

	for i := range metricNames {
		// Save the plots to a file.
		if err := plots[i].Save(6*vg.Inch, 4*vg.Inch, fmt.Sprintf("metric_%s.png", metricNames[i])); err != nil {
			return err
		}
	}

	return nil
}

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
