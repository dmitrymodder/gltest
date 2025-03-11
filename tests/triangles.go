package main

import (
	"encoding/csv"
	"fmt"
	"math/rand"
	"os"
	"runtime"
	"strconv"
	"syscall"
	"time"
	"unsafe"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"
)

const (
	windowWidth  = 1024
	windowHeight = 768
	stageTime    = 10
	totalStages  = 6
	testDuration = 60
)

var (
	vao, vbo, ebo uint32
	shaderProgram uint32
	startTime     time.Time
	particleCounts = []int{10000, 50000, 100000, 500000, 1000000, 10000000}
)

func getWindowsInfo() (string, string) {
    return "Windows", "triangles.csv"
}

func createWindow() *glfw.Window {
    if err := glfw.Init(); err != nil {
        panic(err)
    }
    glfw.WindowHint(glfw.ContextVersionMajor, 4)
    glfw.WindowHint(glfw.ContextVersionMinor, 1)
    glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
    glfw.WindowHint(glfw.Resizable, glfw.False)

    // Create window
    window, err := glfw.CreateWindow(windowWidth, windowHeight, "GLTest | Triangles", nil, nil)
    if err != nil {
        panic(err)
    }
    window.MakeContextCurrent()

    // Center
    monitor := glfw.GetPrimaryMonitor()
    if monitor == nil {
        panic("Failed to get the main monitor")
    }
    mode := monitor.GetVideoMode()
    if mode == nil {
        panic("Failed to get the video mode of the monitor")
    }

    // Get coordinates
    xPos := (mode.Width - windowWidth) / 2
    yPos := (mode.Height - windowHeight) / 2
    window.SetPos(xPos, yPos)

    return window
}

func compileShader(shader uint32, source string) {
	csource, free := gl.Strs(source + "\x00")
	defer free()
	gl.ShaderSource(shader, 1, csource, nil)
	gl.CompileShader(shader)

	var status int32
	gl.GetShaderiv(shader, gl.COMPILE_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetShaderiv(shader, gl.INFO_LOG_LENGTH, &logLength)
		log := make([]byte, logLength)
		gl.GetShaderInfoLog(shader, logLength, nil, &log[0])
		panic(fmt.Errorf("failed to compile shader: %v", string(log)))
	}
}

func notifyWindows(title, message string) {
	user32 := syscall.NewLazyDLL("user32.dll")
	messageBox := user32.NewProc("MessageBoxW")
	titlePtr, _ := syscall.UTF16PtrFromString(title)
	messagePtr, _ := syscall.UTF16PtrFromString(message)
	messageBox.Call(0, uintptr(unsafe.Pointer(messagePtr)), uintptr(unsafe.Pointer(titlePtr)), 0)
}

func createGeometry(numPoints int) ([]float32, []uint32) {
	vertices := make([]float32, 0, numPoints*3)
	indices := make([]uint32, 0, (numPoints/3)*3)

	// Generate random points
	for i := 0; i < numPoints; i++ {
		x := (rand.Float32() - 0.5) * 10.0 // [-5, 5]
		y := (rand.Float32() - 0.5) * 10.0
		z := (rand.Float32() - 0.5) * 10.0
		vertices = append(vertices, x, y, z)
	}

	// Connect point
	for i := 0; i < numPoints-2; i += 3 {
		indices = append(indices, uint32(i), uint32(i+1), uint32(i+2))
	}

	return vertices, indices
}

func initGL() {
	if err := gl.Init(); err != nil {
		panic(err)
	}

	vertexShader := gl.CreateShader(gl.VERTEX_SHADER)
	vertexSource := `#version 410 core
		layout (location = 0) in vec3 position;
		uniform mat4 mvp;
		uniform float time;
		out vec3 fragPos;

		void main() {
			vec3 pos = position;
			gl_Position = mvp * vec4(pos, 1.0);
			fragPos = pos;
		}`
	compileShader(vertexShader, vertexSource)

	fragmentShader := gl.CreateShader(gl.FRAGMENT_SHADER)
	fragmentSource := `#version 410 core
		in vec3 fragPos;
		out vec4 FragColor;

		void main() {
			// Простой цвет на основе позиции
			vec3 color = normalize(fragPos) * 0.5 + 0.5;
			FragColor = vec4(color, 1.0);
		}`
	compileShader(fragmentShader, fragmentSource)

	shaderProgram = gl.CreateProgram()
	gl.AttachShader(shaderProgram, vertexShader)
	gl.AttachShader(shaderProgram, fragmentShader)
	gl.LinkProgram(shaderProgram)

	gl.DeleteShader(vertexShader)
	gl.DeleteShader(fragmentShader)

	gl.GenVertexArrays(1, &vao)
	gl.GenBuffers(1, &vbo)
	gl.GenBuffers(1, &ebo)
}

func drawGeometry(vertices []float32, indices []uint32, currentTime float32) {
    gl.BindVertexArray(vao)

    gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
    gl.BufferData(gl.ARRAY_BUFFER, len(vertices)*4, gl.Ptr(vertices), gl.STATIC_DRAW)

    gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, ebo)
    gl.BufferData(gl.ELEMENT_ARRAY_BUFFER, len(indices)*4, gl.Ptr(indices), gl.STATIC_DRAW)

    gl.EnableVertexAttribArray(0)
    gl.VertexAttribPointer(0, 3, gl.FLOAT, false, 3*4, gl.PtrOffset(0))

    gl.UseProgram(shaderProgram)

    // Вращение фигуры
    projection := mgl32.Perspective(mgl32.DegToRad(45.0), float32(windowWidth)/windowHeight, 0.1, 100.0)
    view := mgl32.LookAtV(mgl32.Vec3{0, 0, 15}, mgl32.Vec3{0, 0, 0}, mgl32.Vec3{0, 1, 0})
    rotation := mgl32.HomogRotate3DY(currentTime * 0.5) // Вращение вокруг Y
    model := rotation
    mvp := projection.Mul4(view).Mul4(model)
    mvpLoc := gl.GetUniformLocation(shaderProgram, gl.Str("mvp\x00"))
    gl.UniformMatrix4fv(mvpLoc, 1, false, &mvp[0])

    timeLoc := gl.GetUniformLocation(shaderProgram, gl.Str("time\x00"))
    gl.Uniform1f(timeLoc, currentTime)

    gl.Enable(gl.DEPTH_TEST)

    gl.DrawElements(gl.TRIANGLES, int32(len(indices)), gl.UNSIGNED_INT, gl.PtrOffset(0))

    gl.DisableVertexAttribArray(0)
}

func checkGLError() {
	if err := gl.GetError(); err != gl.NO_ERROR {
		fmt.Printf("OpenGL error: %d\n", err)
	}
}

func main() {
	runtime.LockOSThread()
	window := createWindow()
	defer glfw.Terminate()
	initGL()
	gl.ClearColor(0.1, 0.1, 0.1, 1.0)

	_, fileName := getWindowsInfo()
	file, err := os.Create(fileName)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush()

	writer.Write([]string{"Time (s)", "Stage", "Points", "Avg FPS", "Min FPS"})

	startTime = time.Now()
	testStart := startTime
	lastRecordTime := testStart
	var frameTimes []float64
	currentStage := 0

	vertices, indices := createGeometry(particleCounts[currentStage])

	for !window.ShouldClose() && time.Since(testStart).Seconds() < testDuration {
		frameStart := time.Now()

		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		currentTime := float32(time.Since(startTime).Seconds())
		drawGeometry(vertices, indices, currentTime)
		window.SwapBuffers()
		glfw.PollEvents()
		checkGLError()

		frameTime := time.Since(frameStart).Seconds()
		frameTimes = append(frameTimes, frameTime)

		timeElapsed := time.Since(testStart).Seconds()
		newStage := int(timeElapsed / stageTime)
		if newStage != currentStage && newStage < len(particleCounts) {
			currentStage = newStage
			fmt.Printf("\nStarting stage %d with %d points\n", currentStage+1, particleCounts[currentStage])
			vertices, indices = createGeometry(particleCounts[currentStage])
		}

		if time.Since(lastRecordTime).Seconds() >= 0.5 {
			var totalFrameTime float64
			for _, ft := range frameTimes {
				totalFrameTime += ft
			}
			avgFPS := float64(len(frameTimes)) / totalFrameTime
			minFPS := 1.0 / maxFrameTime(frameTimes)

			writer.Write([]string{
				strconv.FormatFloat(timeElapsed, 'f', 1, 64),
				strconv.Itoa(currentStage + 1),
				strconv.Itoa(particleCounts[currentStage]),
				strconv.FormatFloat(avgFPS, 'f', 1, 64),
				strconv.FormatFloat(minFPS, 'f', 1, 64),
			})
			writer.Flush()

			fmt.Printf("Time: %.1fs, Stage: %d, Points: %d, Avg FPS: %.1f, Min FPS: %.1f\n",
				timeElapsed, currentStage+1, particleCounts[currentStage], avgFPS, minFPS)

			frameTimes = nil
			lastRecordTime = time.Now()
		}
	}

	//notifyWindows("Geometry Benchmark", "Тест завершен. Данные сохранены в "+fileName)
}

func maxFrameTime(frameTimes []float64) float64 {
	max := frameTimes[0]
	for _, ft := range frameTimes {
		if ft > max {
			max = ft
		}
	}
	return max
}