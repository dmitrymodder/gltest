package main

import (
    "bytes"
    "encoding/csv"
    "encoding/json"
    "fmt"
    "net/http"
    "os"
    "os/exec"
    "path/filepath"
    "regexp"
    "strconv"
    "strings"
    "syscall"
    "time"
    "unsafe"
)

// WinAPI structures
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

// Keys
const (
    SUPABASE_URL = "your_db"
    SUPABASE_KEY = "your_key"
    TABLE_NAME   = "your_table"
)

// FPS structure
type FpsEntry struct {
    Time float64 `json:"time"`
    Fps  float64 `json:"fps"`
}

// Supabase structure
type BenchmarkResult struct {
    GpuName                string      `json:"gpu_name"`
    VramSize               string      `json:"vram_size"`
    DriverVersion          string      `json:"driver_version"`
    WindowsVersion         string      `json:"windows_version"`
    UsesWine               bool        `json:"uses_wine"`
    ButterflyScore         float64     `json:"butterfly_score"`
    TrianglesScore         float64     `json:"triangles_score"`
    OceanScore             float64     `json:"ocean_score"`
    TotalScore             float64     `json:"total_score"`
    ButterflyAvgFps        float64     `json:"butterfly_avg_fps"`
    ButterflyMinFps        float64     `json:"butterfly_min_fps"`
    TrianglesAvgFps        float64     `json:"triangles_avg_fps"`
    TrianglesMinFps        float64     `json:"triangles_min_fps"`
    OceanAvgFps            float64     `json:"ocean_avg_fps"`
    OceanMinFps            float64     `json:"ocean_min_fps"`
    ButterflyAvgFpsHistory []FpsEntry  `json:"butterfly_avg_fps_history"`
    ButterflyMinFpsHistory []FpsEntry  `json:"butterfly_min_fps_history"`
    TrianglesAvgFpsHistory []FpsEntry  `json:"triangles_avg_fps_history"`
    TrianglesMinFpsHistory []FpsEntry  `json:"triangles_min_fps_history"`
    OceanAvgFpsHistory     []FpsEntry  `json:"ocean_avg_fps_history"`
    OceanMinFpsHistory     []FpsEntry  `json:"ocean_min_fps_history"`
    RamSize                string      `json:"ram_size"`
    CpuName                string      `json:"cpu_name"`
    CreatedAt              time.Time   `json:"created_at"`
}

// Get hardware info
func getSystemInfo() (gpuName, vramSize, driverVersion, ramSize, cpuName string) {
    gpuName = "Unknown GPU"
    vramSize = "Unknown"
    driverVersion = "Unknown"
    ramSize = "Unknown"
    cpuName = "Unknown CPU"

    // GPU
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

    // VRAM
    cmdVRAM := exec.Command("wmic", "path", "Win32_VideoController", "get", "AdapterRAM")
    cmdVRAM.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
    var outVRAM bytes.Buffer
    cmdVRAM.Stdout = &outVRAM
    if err := cmdVRAM.Run(); err == nil {
        vramBytes, err := strconv.ParseInt(strings.TrimSpace(string(outVRAM.Bytes()[10:])), 10, 64)
        if err == nil {
            vramMB := vramBytes / (1024 * 1024)
            vramSize = fmt.Sprintf("%d", vramMB)
        }
    }

    // Driver Version
    cmdDriver := exec.Command("wmic", "path", "Win32_VideoController", "get", "DriverVersion")
    cmdDriver.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
    var outDriver bytes.Buffer
    cmdDriver.Stdout = &outDriver
    if err := cmdDriver.Run(); err == nil {
        driverVersion = strings.TrimSpace(string(outDriver.Bytes()[14:]))
    }

    // RAM
    cmdRAM := exec.Command("wmic", "OS", "get", "TotalVisibleMemorySize")
    cmdRAM.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
    var outRAM bytes.Buffer
    cmdRAM.Stdout = &outRAM
    if err := cmdRAM.Run(); err == nil {
        ramKB, err := strconv.ParseInt(strings.TrimSpace(string(outRAM.Bytes()[22:])), 10, 64)
        if err == nil {
            ramMB := ramKB / 1024
            ramSize = fmt.Sprintf("%d", ramMB)
        }
    }

    // CPU
    cmdCPU := exec.Command("wmic", "CPU", "get", "Name")
    cmdCPU.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
    var outCPU bytes.Buffer
    cmdCPU.Stdout = &outCPU
    if err := cmdCPU.Run(); err == nil {
        cpuName = strings.TrimSpace(string(outCPU.Bytes()[5:]))
    }

    return gpuName, vramSize, driverVersion, ramSize, cpuName
}

// Get Windows version
func getWindowsVersion() string {
    cmdEdition := exec.Command("reg", "query", "HKLM\\SOFTWARE\\Microsoft\\Windows NT\\CurrentVersion", "/v", "ProductName")
    cmdEdition.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
    var outEdition bytes.Buffer
    cmdEdition.Stdout = &outEdition

    var edition string
    if err := cmdEdition.Run(); err == nil {
        output := strings.TrimSpace(outEdition.String())
        parts := strings.Split(output, "    ")
        if len(parts) >= 3 {
            edition = strings.TrimSpace(parts[len(parts)-1])
        }
    }

    if edition == "" {
        edition = "Windows"
    }

    cmdBuild := exec.Command("reg", "query", "HKLM\\SOFTWARE\\Microsoft\\Windows NT\\CurrentVersion", "/v", "CurrentBuildNumber")
    cmdBuild.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
    var outBuild bytes.Buffer
    cmdBuild.Stdout = &outBuild

    var build string
    if err := cmdBuild.Run(); err == nil {
        output := strings.TrimSpace(outBuild.String())
        parts := strings.Split(output, "    ")
        if len(parts) >= 3 {
            build = strings.TrimSpace(parts[len(parts)-1])
        }
    }

    cmdUBR := exec.Command("reg", "query", "HKLM\\SOFTWARE\\Microsoft\\Windows NT\\CurrentVersion", "/v", "UBR")
    cmdUBR.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
    var outUBR bytes.Buffer
    cmdUBR.Stdout = &outUBR

    var ubr string
    if err := cmdUBR.Run(); err == nil {
        output := strings.TrimSpace(outUBR.String())
        parts := strings.Split(output, "    ")
        if len(parts) >= 3 {
            ubr = strings.TrimSpace(parts[len(parts)-1])
        }
    }

    var version string
    if build != "" {
        if ubr != "" {
            version = fmt.Sprintf("%s (%s.%s)", edition, build, ubr)
        } else {
            version = fmt.Sprintf("%s (%s)", edition, build)
        }
    } else {
        cmd := exec.Command("cmd", "/c", "ver")
        cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
        var out bytes.Buffer
        cmd.Stdout = &out
        if err := cmd.Run(); err == nil {
            verOutput := strings.TrimSpace(out.String())
            re := regexp.MustCompile(`\[(.*?)\]`)
            matches := re.FindStringSubmatch(verOutput)
            if len(matches) > 1 {
                version = fmt.Sprintf("%s (%s)", edition, strings.TrimPrefix(matches[1], "Version "))
            } else {
                version = edition
            }
        } else {
            version = edition
        }
    }

    return version
}

// Wine
func isWineUsed() bool {
    _, exists := os.LookupEnv("WINEDEBUG")
    return exists
}

// Results structure
type BenchmarkResults struct {
    ButterflyScore float64
    TrianglesScore float64
    OceanScore     float64
    TotalScore     float64
}

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

    for testName, normalizeFactor := range tests {
        csvPath := filepath.Join(testsDir, testName+".csv")

        file, err := os.Open(csvPath)
        if err != nil {
            return results, fmt.Errorf("Failed to open file %s: %v", csvPath, err)
        }
        defer file.Close()

        reader := csv.NewReader(file)
        reader.Comma = ','
        rows, err := reader.ReadAll()
        if err != nil {
            return results, fmt.Errorf("Failed to read CSV %s: %v", csvPath, err)
        }

        if len(rows) <= 1 {
            return results, fmt.Errorf("Insufficient data in file %s", csvPath)
        }
        rows = rows[1:]

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

        rowCount := float64(len(rows))
        if rowCount > 0 {
            avgFps := avgFpsSum / rowCount
            minFps := minFpsSum / rowCount
            avgLoad := avgLoadSum / rowCount

            score := ((avgFps*0.7 + minFps*0.3) * avgLoad) / normalizeFactor

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

    results.TotalScore = results.ButterflyScore + results.TrianglesScore + results.OceanScore

    return results, nil
}

// Parse FPS and Time
func parseFPSResults() (map[string]struct {
    Avg        float64
    Min        float64
    AvgHistory []FpsEntry
    MinHistory []FpsEntry
}, error) {
    fpsResults := make(map[string]struct {
        Avg        float64
        Min        float64
        AvgHistory []FpsEntry
        MinHistory []FpsEntry
    })
    tests := []string{"butterfly", "triangles", "ocean"}

    exePath, err := os.Executable()
    if err != nil {
        return nil, err
    }
    testsDir := filepath.Join(filepath.Dir(exePath), "tests")

    for _, testName := range tests {
        csvPath := filepath.Join(testsDir, testName+".csv")

        file, err := os.Open(csvPath)
        if err != nil {
            return nil, fmt.Errorf("Failed to open file %s: %v", csvPath, err)
        }
        defer file.Close()

        reader := csv.NewReader(file)
        reader.Comma = ','
        rows, err := reader.ReadAll()
        if err != nil {
            return nil, fmt.Errorf("Failed to read CSV %s: %v", csvPath, err)
        }

        if len(rows) <= 1 {
            return nil, fmt.Errorf("Insufficient data in file %s", csvPath)
        }
        rows = rows[1:]

        var avgFpsSum, minFpsSum float64
        var avgFpsHistory, minFpsHistory []FpsEntry
        for _, row := range rows {
            if len(row) < 5 {
                continue
            }

            timeSec, _ := strconv.ParseFloat(row[0], 64)
            avgFps, _ := strconv.ParseFloat(row[3], 64)
            minFps, _ := strconv.ParseFloat(row[4], 64)

            avgFpsSum += avgFps
            minFpsSum += minFps
            avgFpsHistory = append(avgFpsHistory, FpsEntry{Time: timeSec, Fps: avgFps})
            minFpsHistory = append(minFpsHistory, FpsEntry{Time: timeSec, Fps: minFps})
        }

        rowCount := float64(len(rows))
        if rowCount > 0 {
            fpsResults[testName] = struct {
                Avg        float64
                Min        float64
                AvgHistory []FpsEntry
                MinHistory []FpsEntry
            }{
                Avg:        avgFpsSum / rowCount,
                Min:        minFpsSum / rowCount,
                AvgHistory: avgFpsHistory,
                MinHistory: minFpsHistory,
            }
        }
    }
    return fpsResults, nil
}

// Send data to Supabase
func sendToSupabase(data BenchmarkResult) error {
    jsonData, err := json.Marshal(data)
    if err != nil {
        return fmt.Errorf("Error marshaling JSON: %v", err)
    }

    url := fmt.Sprintf("%s/rest/v1/%s", SUPABASE_URL, TABLE_NAME)
    req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
    if err != nil {
        return fmt.Errorf("Error creating request: %v", err)
    }

    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("apikey", SUPABASE_KEY)
    req.Header.Set("Prefer", "return=minimal")

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return fmt.Errorf("Error sending request: %v", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
        return fmt.Errorf("HTTP error: %s", resp.Status)
    }

    return nil
}

// Save results to local file
func saveToLocalFile(data BenchmarkResult) error {
    exePath, err := os.Executable()
    if err != nil {
        return fmt.Errorf("Error determining path: %v", err)
    }

    outputPath := filepath.Join(filepath.Dir(exePath), "send.txt")
    file, err := os.Create(outputPath)
    if err != nil {
        return fmt.Errorf("Error creating file: %v", err)
    }
    defer file.Close()

    lines := []string{
        fmt.Sprintf("gpu_name=%s", data.GpuName),
        fmt.Sprintf("vram_size=%s", data.VramSize),
        fmt.Sprintf("driver_version=%s", data.DriverVersion),
        fmt.Sprintf("windows_version=%s", data.WindowsVersion),
        fmt.Sprintf("uses_wine=%t", data.UsesWine),
        fmt.Sprintf("butterfly_score=%.2f", data.ButterflyScore),
        fmt.Sprintf("triangles_score=%.2f", data.TrianglesScore),
        fmt.Sprintf("ocean_score=%.2f", data.OceanScore),
        fmt.Sprintf("total_score=%.2f", data.TotalScore),
        fmt.Sprintf("butterfly_avg_fps=%.2f", data.ButterflyAvgFps),
        fmt.Sprintf("butterfly_min_fps=%.2f", data.ButterflyMinFps),
        fmt.Sprintf("triangles_avg_fps=%.2f", data.TrianglesAvgFps),
        fmt.Sprintf("triangles_min_fps=%.2f", data.TrianglesMinFps),
        fmt.Sprintf("ocean_avg_fps=%.2f", data.OceanAvgFps),
        fmt.Sprintf("ocean_min_fps=%.2f", data.OceanMinFps),
        fmt.Sprintf("butterfly_avg_fps_history=%v", data.ButterflyAvgFpsHistory),
        fmt.Sprintf("butterfly_min_fps_history=%v", data.ButterflyMinFpsHistory),
        fmt.Sprintf("triangles_avg_fps_history=%v", data.TrianglesAvgFpsHistory),
        fmt.Sprintf("triangles_min_fps_history=%v", data.TrianglesMinFpsHistory),
        fmt.Sprintf("ocean_avg_fps_history=%v", data.OceanAvgFpsHistory),
        fmt.Sprintf("ocean_min_fps_history=%v", data.OceanMinFpsHistory),
        fmt.Sprintf("ram_size=%s", data.RamSize),
        fmt.Sprintf("cpu_name=%s", data.CpuName),
        fmt.Sprintf("created_at=%s", data.CreatedAt.Format(time.RFC3339)),
    }

    for _, line := range lines {
        if _, err := file.WriteString(line + "\n"); err != nil {
            return fmt.Errorf("Error writing to file: %v", err)
        }
    }

    return nil
}

func main() {
    if len(os.Args) < 5 {
        exec.Command("msg", "*", "Error: Insufficient arguments for send.exe").Run()
        return
    }

    butterflyScore, _ := strconv.ParseFloat(os.Args[1], 64)
    trianglesScore, _ := strconv.ParseFloat(os.Args[2], 64)
    oceanScore, _ := strconv.ParseFloat(os.Args[3], 64)
    totalScore, _ := strconv.ParseFloat(os.Args[4], 64)

    _ = os.Args[5] // gpuName
    _ = os.Args[6] // vramSize
    _ = os.Args[7] // openGLVersion

    gpuName, vramSize, driverVersion, ramSize, cpuName := getSystemInfo()
    windowsVersion := getWindowsVersion()
    usesWine := isWineUsed()

    fpsResults, err := parseFPSResults()
    if err != nil {
        exec.Command("msg", "*", fmt.Sprintf("Error calculating FPS: %v", err)).Run()
        return
    }

    benchmarkData := BenchmarkResult{
        GpuName:                gpuName,
        VramSize:               vramSize,
        DriverVersion:          driverVersion,
        WindowsVersion:         windowsVersion,
        UsesWine:               usesWine,
        ButterflyScore:         butterflyScore,
        TrianglesScore:         trianglesScore,
        OceanScore:             oceanScore,
        TotalScore:             totalScore,
        ButterflyAvgFps:        fpsResults["butterfly"].Avg,
        ButterflyMinFps:        fpsResults["butterfly"].Min,
        TrianglesAvgFps:        fpsResults["triangles"].Avg,
        TrianglesMinFps:        fpsResults["triangles"].Min,
        OceanAvgFps:            fpsResults["ocean"].Avg,
        OceanMinFps:            fpsResults["ocean"].Min,
        ButterflyAvgFpsHistory: fpsResults["butterfly"].AvgHistory,
        ButterflyMinFpsHistory: fpsResults["butterfly"].MinHistory,
        TrianglesAvgFpsHistory: fpsResults["triangles"].AvgHistory,
        TrianglesMinFpsHistory: fpsResults["triangles"].MinHistory,
        OceanAvgFpsHistory:     fpsResults["ocean"].AvgHistory,
        OceanMinFpsHistory:     fpsResults["ocean"].MinHistory,
        RamSize:                ramSize,
        CpuName:                cpuName,
        CreatedAt:              time.Now(),
    }

    if err := saveToLocalFile(benchmarkData); err != nil {
        exec.Command("msg", "*", fmt.Sprintf("Error saving locally: %v", err)).Run()
    }

    if err := sendToSupabase(benchmarkData); err != nil {
        exec.Command("msg", "*", fmt.Sprintf("Error sending to Supabase: %v\nResults saved locally in send.txt", err)).Run()
        return
    }

    exec.Command("msg", "*", "Benchmark results successfully sent to the database").Run()
}
