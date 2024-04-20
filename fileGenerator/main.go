package main

import (
	"bufio"
	"fmt"
	"github.com/google/uuid"
	"gonum.org/v1/plot/vg"
	"image/color"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"time"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
)

type Disk string

const (
	fileSize        = 10 * 1024 * 1024 // 10MB file size
	bufferSize      = 1024 * 1024      // 4KB buffer size
	ssd        Disk = "Samsung SSD 990 Pro"
	hd         Disk = "Seagate Barracuda ST2000DM008"
)

var (
	numFiles    = 100
	startWrite  time.Time
	startRead   time.Time
	startDelete time.Time
)

type BenchmarkResult struct {
	disk           Disk
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

func main() {
	emp := BenchmarkResult{
		disk:         ssd,
		targetFolder: "D:/junkFiles",
		color:        color.RGBA{R: 0, G: 0, B: 255, A: 255},
	}

	slow := BenchmarkResult{
		disk:         hd,
		targetFolder: "E:/junkFiles",
		color:        color.Black,
	}
	results := make([]BenchmarkResult, 2)
	results[0] = emp
	results[1] = slow
	for j := range results {
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
		if err := os.MkdirAll(results[j].targetFolder, os.ModePerm); err != nil {
			fmt.Println("Error creating target folder:", err)
			return
		}

		// Define file paths for benchmarking
		filePaths := make([]string, 0, numFiles)
		for i := 0; i < numFiles; i++ {
			filePath := filepath.Join(results[j].targetFolder, fmt.Sprintf("file%s.dat", uuid.New()))
			filePaths = append(filePaths, filePath)
		}

		fmt.Println("Starting IO benchmark on " + results[j].disk + "...")

		// Start CPU profile
		cpuProfile, err := os.Create(string("cpu_" + results[j].disk + "_profile.prof"))
		if err != nil {
			fmt.Println("Error creating CPU profile:", err)
			return
		}

		if err := pprof.StartCPUProfile(cpuProfile); err != nil {
			fmt.Println("Error starting CPU profile:", err)
			return
		}
		totalTime := time.Now()
		startWrite = time.Now()
		// Collect benchmark results
		for _, path := range filePaths {
			writeSpeed, writeIOPS, cpuTime, writeTime, err := measureWriteBenchmark(path)
			if err != nil {
				fmt.Printf("Error writing file on %s: %v\n", path, err)
				return
			}

			results[j].writeSpeeds = append(results[j].writeSpeeds, writeSpeed)
			totalWriteSpeed += writeSpeed
			results[j].writeIOPSs = append(results[j].writeIOPSs, writeIOPS)
			totalWriteIOPS += writeIOPS
			results[j].cpuTimes = append(results[j].cpuTimes, cpuTime)
			totalCPUTime += cpuTime
			results[j].writeTimes = append(results[j].writeTimes, writeTime)
		}
		stopWriteTime := time.Now()
		writeDuration := stopWriteTime.Sub(startWrite)

		startRead = time.Now()
		for _, path := range filePaths {
			readSpeed, readIOPS, _, memUsage, readTime, err := measureReadBenchmark(path)
			if err != nil {
				fmt.Printf("Error reading file on %s: %v\n", path, err)
				return
			}

			results[j].readSpeeds = append(results[j].readSpeeds, readSpeed)
			totalReadSpeed += readSpeed
			results[j].readIOPSs = append(results[j].readIOPSs, readIOPS)
			totalReadIOPS += readIOPS
			results[j].memUsages = append(results[j].memUsages, memUsage)
			totalMemUsage += memUsage
			results[j].readTimes = append(results[j].readTimes, readTime)

		}

		stopReadTime := time.Now()
		readDuration := stopReadTime.Sub(startRead)

		startDelete = time.Now()
		for _, path := range filePaths {
			deleteSpeed, cpuTime, deleteTime, err := measureDeleteBenchmark(path)
			if err != nil {
				fmt.Printf("Error reading file on %s: %v\n", path, err)
				return
			}

			results[j].deleteSpeeds = append(results[j].deleteSpeeds, deleteSpeed)
			totalDeleteSpeed += deleteSpeed
			results[j].cpuDeleteTimes = append(results[j].cpuDeleteTimes, cpuTime)
			totalDeleteCPUTime += cpuTime
			results[j].deleteTimes = append(results[j].deleteTimes, deleteTime)
		}

		stopDeleteTime := time.Now()
		deleteDuration := stopDeleteTime.Sub(startDelete)
		cpuProfile.Close()
		pprof.StopCPUProfile()

		// Calculate averages
		avgWriteSpeed := float64(numFiles) * 10 / writeDuration.Seconds()
		avgReadSpeed := float64(numFiles) * 10 / readDuration.Seconds()
		totalWriteReadOps := (fileSize * numFiles) / bufferSize
		avgWriteIOPS := float64(totalWriteReadOps) / writeDuration.Seconds()
		avgReadIOPS := float64(totalWriteReadOps) / readDuration.Seconds()
		avgCPUTime := totalCPUTime / float64(numFiles)
		avgMemUsage := totalMemUsage / float64(numFiles)
		avgDeleteSpeed := float64(numFiles) * 10 / deleteDuration.Seconds()
		avgDeleteIOPS := float64(numFiles) / deleteDuration.Seconds()
		avgDeleteCPUTime := totalDeleteCPUTime / float64(numFiles)

		// Print statistics
		fmt.Println("Disk type:", results[j].disk)
		fmt.Println("Total operation Read, Write, Delete in seconds", time.Since(totalTime).Seconds())
		fmt.Println("Number of files:", numFiles)
		fmt.Println("File size: (KB)", fileSize/1024)
		fmt.Println("Total File size: (MB) ", fileSize*numFiles/(1024*1024))
		fmt.Println("Average Write Speed (MB/s):", avgWriteSpeed)
		fmt.Println("Total Write time in seconds:", writeDuration.Seconds())
		fmt.Println("Average Read Speed (MB/s):", avgReadSpeed)
		fmt.Println("Total Read time in seconds:", readDuration.Seconds())
		fmt.Println("Average Delete Speed (MB/s):", avgDeleteSpeed)
		fmt.Println("Total Delete time in seconds:", deleteDuration.Seconds())
		fmt.Println("Average Write IOPS:", avgWriteIOPS)
		fmt.Println("Average Read IOPS:", avgReadIOPS)
		fmt.Println("Average Delete IOPS:", avgDeleteIOPS)
		fmt.Println("Average CPU Time (s):", avgCPUTime)
		fmt.Println("Average Delete CPU Time (s):", avgDeleteCPUTime)
		fmt.Println("Average Memory Usage (bytes):", avgMemUsage)
		fmt.Printf("\n\n")
	}

	// Plot the metrics
	if err := plotMetrics(results); err != nil {
		fmt.Println("Error plotting metrics:", err)
		return
	}
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
		deleteSpeed = float64(fileSize) / deleteTime / (1024 * 1024)
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
	writeSpeed := float64(fileSize) / duration / (1024 * 1024)

	// Calculate write IOPS
	writeIOPS := float64(fileSize) / float64(bufferSize) / duration

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
	readIOPS := float64(bytesRead) / float64(bufferSize) / duration.Seconds()

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

	buf := make([]byte, bufferSize)
	buf[len(buf)-1] = '\n'
	w := bufio.NewWriterSize(file, len(buf))

	start := time.Now()
	written := int64(0)
	for i := int64(0); i < fileSize; i += int64(len(buf)) {
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
	buffer := make([]byte, bufferSize)

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
func plotMetrics(results []BenchmarkResult) error {
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
			plots[i].Legend.Add(string(result.disk), line)

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
			plots[i].Legend.Add(string(result.disk), line)

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
			plots[i].Legend.Add(string(result.disk), line)

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
