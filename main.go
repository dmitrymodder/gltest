package main

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"fyne.io/fyne/v2"
    "fyne.io/fyne/v2/app"
    "fyne.io/fyne/v2/container"
    "fyne.io/fyne/v2/dialog"
    "fyne.io/fyne/v2/layout"
    "fyne.io/fyne/v2/widget"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"syscall"
	"strings"
	"time"
	"unsafe"
)

// Windows API structures
type DISPLAY_DEVICE struct {
	cb           uint32
	DeviceName   [32]uint16
	DeviceString [128]uint16
	StateFlags   uint32
	DeviceID     [128]uint16
	DeviceKey    [128]uint16
}

const (
	DISPLAY_DEVICE_PRIMARY_DEVICE = 0x00000004
)

func getGPUInfo() (gpuName string, vramSize string, driverVersion string) {
	gpuName = "Unknown GPU"
	vramSize = "Unknown VRAM"
	driverVersion = "Unknown Driver"

	// Get GPU name from Windows API
	var dd DISPLAY_DEVICE
	dd.cb = uint32(unsafe.Sizeof(dd))
	user32 := syscall.NewLazyDLL("user32.dll")
	enumDisplayDevices := user32.NewProc("EnumDisplayDevicesW")

	for i := uint32(0); ; i++ {
		ret, _, _ := enumDisplayDevices.Call(0, uintptr(i), uintptr(unsafe.Pointer(&dd)), 0)
		if ret == 0 {
			break
		}
		if dd.StateFlags&DISPLAY_DEVICE_PRIMARY_DEVICE != 0 {
			gpuName = syscall.UTF16ToString(dd.DeviceString[:])
			break
		}
	}

	// Get VRAM with wmic
	cmdVRAM := exec.Command("wmic", "path", "Win32_VideoController", "get", "AdapterRAM")
	// Hide process
	cmdVRAM.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	var outVRAM bytes.Buffer
	cmdVRAM.Stdout = &outVRAM
	if err := cmdVRAM.Run(); err == nil {
		vramBytes, err := strconv.ParseInt(strings.TrimSpace(string(outVRAM.Bytes()[10:])), 10, 64)
		if err == nil {
			vramMB := vramBytes / (1024 * 1024)
			vramSize = fmt.Sprintf("%d MB VRAM", vramMB)
		}
	}

	// Get Driver version with wmic
	cmdDriver := exec.Command("wmic", "path", "Win32_VideoController", "get", "DriverVersion")
	cmdDriver.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	var outDriver bytes.Buffer
	cmdDriver.Stdout = &outDriver
	if err := cmdDriver.Run(); err == nil {
		driverVersion = "Driver: " + strings.TrimSpace(string(outDriver.Bytes()[14:]))
	}

	return gpuName, vramSize, driverVersion
}

// Results structure
type BenchmarkResults struct {
	ButterflyScore float64
	TrianglesScore float64
	OceanScore     float64
	TotalScore     float64
}

// Run tests
func runTest(testName string) error {
	exePath, err := os.Executable()
	if err != nil {
		return err
	}
	testsDir := filepath.Join(filepath.Dir(exePath), "tests")
	
	// Run benchmark
	cmd := exec.Command(filepath.Join(testsDir, testName+".exe"))
	cmd.Dir = testsDir
	
	// Start testing process
	if err := cmd.Start(); err != nil {
		return err
	}
	
	return cmd.Wait()
}

// Analyze CSV
func parseResultsAndCalculateScore() (BenchmarkResults, error) {
	results := BenchmarkResults{}
	
	tests := map[string]float64{
		"butterfly": 16384000,
		"ocean":     6,
		"triangles": 10000000,
	}
	
	exePath, err := os.Executable()
	if err != nil {
		return results, err
	}
	testsDir := filepath.Join(filepath.Dir(exePath), "tests")
	
	// Tests processing
	for testName, normalizeFactor := range tests {
		csvPath := filepath.Join(testsDir, testName+".csv")
		
		// Opening CSV
		file, err := os.Open(csvPath)
		if err != nil {
			return results, fmt.Errorf("Failed to open file %s: %v", csvPath, err)
		}
		defer file.Close()
		
		// Reading CSV
		reader := csv.NewReader(file)
		reader.Comma = ',' 
		rows, err := reader.ReadAll()
		if err != nil {
			return results, fmt.Errorf("Error reading CSV %s: %v", csvPath, err)
		}
		
		// Skip header
		if len(rows) <= 1 {
			return results, fmt.Errorf("Insufficient data in file %s", csvPath)
		}
		rows = rows[1:]
		
		// Calculate average
		var avgFpsSum, minFpsSum, avgLoadSum float64
		for _, row := range rows {
			if len(row) < 5 {
				continue
			}
			
			avgFps, _ := strconv.ParseFloat(row[3], 64)
			minFps, _ := strconv.ParseFloat(row[4], 64)
			particles, _ := strconv.ParseFloat(row[2], 64)
			
			avgFpsSum += avgFps
			minFpsSum += minFps
			avgLoadSum += particles
		}
		
		// Calculate
		rowCount := float64(len(rows))
		if rowCount > 0 {
			avgFps := avgFpsSum / rowCount
			minFps := minFpsSum / rowCount
			avgLoad := avgLoadSum / rowCount
			
			// Formula
			score := ((avgFps*0.7 + minFps*0.3) * avgLoad) / normalizeFactor
			
			// Save result
			switch testName {
			case "butterfly":
				results.ButterflyScore = score
			case "triangles":
				results.TrianglesScore = score
			case "ocean":
				results.OceanScore = score
			}
		}
	}
	
	// Calculate final result
	results.TotalScore = results.ButterflyScore + results.TrianglesScore + results.OceanScore
	
	return results, nil
}

// Send result to server
func sendResultsForStatistics(results BenchmarkResults, gpuInfo string) error {
	log.Println("Sending results to server:", results)
	return nil
}

func main() {
	runtime.LockOSThread()
	
	// Create Fyne App
	a := app.New()
	w := a.NewWindow("GLTest")
	w.Resize(fyne.NewSize(300, 400))
	
	// Get GPU info
	gpuName, vramSize, openGLVersion := getGPUInfo()

	// Create UI elements
	// GPU Information
	gpuNameLabel := widget.NewLabel(gpuName)
	vramSizeLabel := widget.NewLabel(vramSize)
	openGLVersionLabel := widget.NewLabel(openGLVersion)
	gpuInfoContainer := container.NewVBox(
		widget.NewLabelWithStyle("My GPU", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		gpuNameLabel,
		vramSizeLabel,
		openGLVersionLabel,
	)
	
	// Results section
	butterflyScore := widget.NewLabel("-")
	trianglesScore := widget.NewLabel("-")
	oceanScore := widget.NewLabel("-")
	totalScore := widget.NewLabel("-")
	
	// Create results section
	resultsGrid := container.New(layout.NewGridLayout(2),
		widget.NewLabel("Butterfly"),
		butterflyScore,
		widget.NewLabel("Triangles"),
		trianglesScore,
		widget.NewLabel("Ocean"),
		oceanScore,
		widget.NewLabel("Total"),
		totalScore,
	)
	
	resultsContainer := container.NewVBox(
		widget.NewLabelWithStyle("Results", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		resultsGrid,
	)
	
	// Send data checkbox
	sendStatsCheck := widget.NewCheck("Send results for statistics (recommended)", nil)
	sendStatsCheck.SetChecked(true)
	
	// Start button
startButton := widget.NewButton("Start Benchmark", nil)
startButton.OnTapped = func() {
    startButton.Disable()

    // Reset results
    butterflyScore.SetText("-")
    trianglesScore.SetText("-")
    oceanScore.SetText("-")
    totalScore.SetText("-")

    go func() {
        tests := []string{"butterfly", "triangles", "ocean"}

        for _, test := range tests {
            err := runTest(test)
            if err != nil {
                log.Printf("Failed to run test %s: %v", test, err)
                dialog.ShowError(fmt.Errorf("Failed to run test %s: %v", test, err), w)
                startButton.Enable()
                return
            }
            time.Sleep(500 * time.Millisecond)
        }

        results, err := parseResultsAndCalculateScore()
        if err != nil {
            log.Printf("Failed to analyze results: %v", err)
            dialog.ShowError(fmt.Errorf("Failed to analyze results: %v", err), w)
            startButton.Enable()
            return
        }

        // Update UI
        butterflyScore.SetText(fmt.Sprintf("%.2f", results.ButterflyScore))
        trianglesScore.SetText(fmt.Sprintf("%.2f", results.TrianglesScore))
        oceanScore.SetText(fmt.Sprintf("%.2f", results.OceanScore))
        totalScore.SetText(fmt.Sprintf("%.2f", results.TotalScore))

        // Start send process
        if sendStatsCheck.Checked {
            exePath, err := os.Executable()
            if err != nil {
                exec.Command("msg", "*", fmt.Sprintf("Error: Failed to determine executable path: %v", err)).Run()
                startButton.Enable()
                return
            }
            sendPath := filepath.Join(filepath.Dir(exePath), "send.exe")
            cmd := exec.Command(sendPath,
                fmt.Sprintf("%f", results.ButterflyScore),
                fmt.Sprintf("%f", results.TrianglesScore),
                fmt.Sprintf("%f", results.OceanScore),
                fmt.Sprintf("%f", results.TotalScore),
                gpuName, vramSize, openGLVersion)
            cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
            if err := cmd.Run(); err != nil {
                exec.Command("msg", "*", fmt.Sprintf("Error running send.exe: %v", err)).Run()
            }
        }

        startButton.Enable()
    }()
}
	
	// Create main container
	content := container.NewVBox(
		gpuInfoContainer,
		widget.NewSeparator(),
		resultsContainer,
		widget.NewSeparator(),
		sendStatsCheck,
		startButton,
	)
	
	// Open window
	w.SetContent(content)
	w.ShowAndRun()
}