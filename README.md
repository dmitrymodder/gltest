# GLTest - OpenGL Benchmark Tool

![GitHub release (latest by date)](https://img.shields.io/github/v/release/dmitrymodder/gltest?style=flat-square)
![GitHub license](https://img.shields.io/github/license/dmitrymodder/gltest?style=flat-square)

GLTest is a tool for testing graphics processing unit (GPU) performance using OpenGL. The project is designed to evaluate a video card's capabilities through a series of tests, including rendering millions of dots, triangles and wave simulations, and then calculating a final score. The results can be sent to your database, such as Supabase.

## What is it?

GLTest is a benchmark designed for Windows that:
- Determines GPU characteristics (name, VRAM capacity, driver version).
- Performs three performance tests: `butterfly`, `triangles`, `ocean`.
- Calculates a final score based on average and minimum FPS, as well as load.
- Provides a graphical interface based on the Fyne library.
- Supports sending results for statistics via a separate executable file `send.exe`.

The project is written in Go and uses OpenGL for rendering tests.
## What include?

- **main.go**: Main application file with GUI and test run logic.
- **send.go**: Utility to send results to the server (Supabase)
- **tests/**: Directory with tests:
  - `butterfly.go` - test of rendering a set of points as an infinity sign.
  - `triangles.go` - test of rendering triangles.
  - `ocean.go` - test of wave simulation.
- **Makefile**: Script for automated project build.
- **build/**: Output directory of the build (created automatically).

## When I can view my results?
If you used the release version, all results can be found at https://gltestsite.vercel.app/.

## System requirements

- To work: OpenGL support, 64 bit OS
- To build: MSYS2, golang, make

## How to build?

The project uses `Makefile` to automate the build and requires Go, `gcc` (for `cgo`) and `make`. Below are the instructions for Windows.

### Requirements
- OS: Windows 10/11.
- Golang
- C compiler (gcc)

#### Install dependencies
1. **Go**:
   - Download and install from [golang.org/dl/](https://golang.org/dl/).
   - Check: `go version` in the terminal.
2. **MSYS2 and MinGW** (for `gcc`):
   - Install MSYS2 from [msys2.org](https://www.msys2.org/)
   - In the MSYS2 terminal, execute:
>pacman -Syu\
>pacman -S mingw-w64-x86_64-gcc
- Add `C:\Program Files\msys64\mingw64\bin` to PATH.
- Check: `gcc --version`.
3. **Make**:
- Install Chocolatey (if not installed): [manual](https://chocolatey.org/install).
- Run: `choco install make`.
- Check: `make --version`.
4. **Go** Dependencies:
- Install the required packages:
>fyne.io/fyne/v2/*  
>github.com/go-gl/gl/v4.1-core/gl  
>github.com/go-gl/glfw/v3.3/glfw
### Assembly
1. Clone the repository
2. Perform the build:
This will create `build/GLTest.exe`, `build/tests/*.exe`, and `build/send.exe`.
### Run
- Go to `build` and run: `GLTest.exe`.
