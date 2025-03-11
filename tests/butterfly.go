package main

import (
	"encoding/csv"
	"fmt"
	"math"
	"math/rand"
	"os"
	"runtime"
	"strconv"
	"syscall"
	"time"
	"unsafe"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
)

const (
	windowWidth  = 1024
	windowHeight = 768
	stageTime    = 10
	totalStages  = 12
	testDuration = 120
	warmUpTime   = 2
)

type Color struct {
	r, g, b, a float32
}

type Particle struct {
	baseX, baseY float32
	phase        float32
	distance     float32
	wingPos      float32
	size         float32
}

var (
	vao, vbo      uint32
	shaderProgram uint32
	startTime     time.Time
	particleCounts = []int{
		8000,
		16000,
		32000,
		64000,
		128000,
		256000,
		512000,
		1024000,
		2048000,
		4096000,
		8192000,
		16384000,
	}
	colors = []Color{
		{1.0, 0.0, 0.0, 1.0}, // Red
		{1.0, 0.5, 0.0, 1.0}, // Orange
		{1.0, 1.0, 0.0, 1.0}, // Yellow
		{0.0, 1.0, 0.0, 1.0}, // Green
		{0.0, 1.0, 1.0, 1.0}, // Cyan
		{0.0, 0.0, 1.0, 1.0}, // Blue
		{0.5, 0.0, 1.0, 1.0}, // Purple
		{1.0, 0.0, 1.0, 1.0}, // Magenta
	}
)

func getWindowsInfo() (string, string) {
    return "Windows", "butterfly.csv"
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
    window, err := glfw.CreateWindow(windowWidth, windowHeight, "GLTest | Butterfly", nil, nil)
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

func createButterflyParticles(count int) []Particle {
	particles := make([]Particle, count)
	for i := range particles {
		angle := rand.Float32() * 2 * math.Pi
		distance := rand.Float32() * 0.8
		particles[i] = Particle{
			baseX:    float32(math.Cos(float64(angle))) * distance,
			baseY:    float32(math.Sin(float64(angle))) * distance,
			phase:    rand.Float32() * 2 * math.Pi,
			distance: distance,
			wingPos:  rand.Float32() * math.Pi,
			size:     3.0 + rand.Float32()*2.0,
		}
	}
	return particles
}

func lerpColor(c1, c2 Color, t float32) Color {
	return Color{
		r: c1.r + (c2.r-c1.r)*t,
		g: c1.g + (c2.g-c1.g)*t,
		b: c1.b + (c2.b-c1.b)*t,
		a: c1.a + (c2.a-c1.a)*t,
	}
}

func drawParticles(particles []Particle, currentTime float32) {
	data := make([]float32, len(particles)*6) // baseX, baseY, phase, distance, wingPos, size
	for i, p := range particles {
		base := i * 6
		data[base] = p.baseX
		data[base+1] = p.baseY
		data[base+2] = p.phase
		data[base+3] = p.distance
		data[base+4] = p.wingPos
		data[base+5] = p.size
	}

	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE)

	gl.BindVertexArray(vao)
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
	gl.BufferData(gl.ARRAY_BUFFER, len(data)*4, gl.Ptr(data), gl.STATIC_DRAW)

	gl.UseProgram(shaderProgram)

	// Передаем uniform-переменные
	timeLoc := gl.GetUniformLocation(shaderProgram, gl.Str("time\x00"))
	gl.Uniform1f(timeLoc, currentTime)

	colorIndex := int(currentTime/2.0) % len(colors)
	nextColorIndex := (colorIndex + 1) % len(colors)
	colorT := float32(math.Mod(float64(currentTime/2.0), 1.0))
	currentColor := lerpColor(colors[colorIndex], colors[nextColorIndex], colorT)
	colorLoc := gl.GetUniformLocation(shaderProgram, gl.Str("currentColor\x00"))
	gl.Uniform4f(colorLoc, currentColor.r, currentColor.g, currentColor.b, currentColor.a)

	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointer(0, 2, gl.FLOAT, false, 6*4, gl.PtrOffset(0)) // baseX, baseY
	gl.EnableVertexAttribArray(1)
	gl.VertexAttribPointer(1, 1, gl.FLOAT, false, 6*4, gl.PtrOffset(2*4)) // phase
	gl.EnableVertexAttribArray(2)
	gl.VertexAttribPointer(2, 1, gl.FLOAT, false, 6*4, gl.PtrOffset(3*4)) // distance
	gl.EnableVertexAttribArray(3)
	gl.VertexAttribPointer(3, 1, gl.FLOAT, false, 6*4, gl.PtrOffset(4*4)) // wingPos
	gl.EnableVertexAttribArray(4)
	gl.VertexAttribPointer(4, 1, gl.FLOAT, false, 6*4, gl.PtrOffset(5*4)) // size

	gl.DrawArrays(gl.POINTS, 0, int32(len(particles)))

	gl.DisableVertexAttribArray(0)
	gl.DisableVertexAttribArray(1)
	gl.DisableVertexAttribArray(2)
	gl.DisableVertexAttribArray(3)
	gl.DisableVertexAttribArray(4)
}

func initGL() {
	if err := gl.Init(); err != nil {
		panic(err)
	}

	vertexShader := gl.CreateShader(gl.VERTEX_SHADER)
	vertexSource := `#version 410 core
		layout (location = 0) in vec2 basePos;
		layout (location = 1) in float phase;
		layout (location = 2) in float distance;
		layout (location = 3) in float wingPos;
		layout (location = 4) in float size;
		uniform float time;
		uniform vec4 currentColor;
		out vec4 fragColor;

		void main() {
			float butterflyScale = 0.8;
			float wingSpeed = 3.0;
			float t = phase + time * 0.5;
			float scale = distance;

			// Basic butterfly movement
			vec2 pos;
			pos.x = butterflyScale * scale * sin(t);
			pos.y = butterflyScale * scale * sin(t) * cos(t);

			// Wing beats
			float wingOffset = sin(time * wingSpeed + wingPos);
			pos.x += wingOffset * scale * 0.2;

			gl_Position = vec4(pos, 0.0, 1.0);
			fragColor = currentColor;
			gl_PointSize = size;
		}`
	compileShader(vertexShader, vertexSource)

	fragmentShader := gl.CreateShader(gl.FRAGMENT_SHADER)
	fragmentSource := `#version 410 core
		in vec4 fragColor;
		out vec4 FragColor;
		void main() {
			vec2 circCoord = 2.0 * gl_PointCoord - 1.0;
			float circShape = 1.0 - length(circCoord);
			float alpha = smoothstep(0.0, 1.0, circShape);
			FragColor = vec4(fragColor.rgb, fragColor.a * alpha);
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
	gl.ClearColor(0, 0, 0, 1)

	_, fileName := getWindowsInfo()
	file, err := os.Create(fileName)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush()

	writer.Write([]string{"Time (s)", "Stage", "Particles", "Avg FPS", "Min FPS"})

	// Warming
	fmt.Println("Warming up...")
	warmUpStart := time.Now()
	particles := createButterflyParticles(particleCounts[0])
	for time.Since(warmUpStart).Seconds() < warmUpTime {
		gl.Clear(gl.COLOR_BUFFER_BIT)
		drawParticles(particles, float32(time.Since(warmUpStart).Seconds()))
		window.SwapBuffers()
		glfw.PollEvents()
		checkGLError()
	}

	// Main test
	startTime = time.Now()
	testStart := startTime
	lastRecordTime := testStart
	var frameTimes []float64
	currentStage := 0

	gl.Enable(gl.PROGRAM_POINT_SIZE)

	for !window.ShouldClose() && time.Since(testStart).Seconds() < testDuration {
		frameStart := time.Now()

		gl.Clear(gl.COLOR_BUFFER_BIT)
		currentTime := float32(time.Since(startTime).Seconds())
		drawParticles(particles, currentTime)
		window.SwapBuffers()
		glfw.PollEvents()
		checkGLError()

		frameTime := time.Since(frameStart).Seconds()
		frameTimes = append(frameTimes, frameTime)

		timeElapsed := time.Since(testStart).Seconds()
		newStage := int(timeElapsed / stageTime)
		if newStage != currentStage && newStage < len(particleCounts) {
			currentStage = newStage
			particles = createButterflyParticles(particleCounts[currentStage])
			fmt.Printf("\nStarting stage %d with %d particles\n", currentStage+1, len(particles))
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
				strconv.Itoa(len(particles)),
				strconv.FormatFloat(avgFPS, 'f', 1, 64),
				strconv.FormatFloat(minFPS, 'f', 1, 64),
			})
			writer.Flush()

			fmt.Printf("Time: %.1fs, Stage: %d, Particles: %d, Avg FPS: %.1f, Min FPS: %.1f\n",
				timeElapsed, currentStage+1, len(particles), avgFPS, minFPS)

			frameTimes = nil
			lastRecordTime = time.Now()
		}
	}

	//notifyWindows(“Butterfly GPU Benchmark”, “Test completed. Data saved in ”+fileName)
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
