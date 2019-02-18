package main

import (
	"fmt"
	"log"
	"math"
	"math/cmplx"
	"runtime"
	"strings"
	"time"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.2/glfw"
	"github.com/mjibson/go-dsp/fft"
)

const (
	width  = 500
	height = 500

	vertexShaderSource = `
		#version 410
		in vec3 vp;
		void main() {
			gl_Position = vec4(vp, 1.0);
		}
	` + "\x00"

	fragmentShaderSource = `
		#version 410
		out vec4 frag_colour;
		void main() {
			frag_colour = vec4(1, 0, 0, 1.0);
		}
	` + "\x00"

	rows    = 32
	columns = 32
)

var (
	square = []float32{
		-0.5, 0.5, 0,
		-0.5, -0.5, 0,
		0.5, -0.5, 0,

		-0.5, 0.5, 0,
		0.5, 0.5, 0,
		0.5, -0.5, 0,
	}
)

type cell struct {
	drawable uint32

	x int
	y int
}

func draw(cells [][]*cell, heights []int, window *glfw.Window, program uint32) {
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	gl.UseProgram(program)

	for x := range cells {
		for i, c := range cells[x] {
			if i < heights[x] {
				c.draw()
			}
		}
	}

	glfw.PollEvents()
	window.SwapBuffers()
}

func makeCells() [][]*cell {
	cells := make([][]*cell, rows, rows)
	for x := 0; x < rows; x++ {
		for y := 0; y < columns; y++ {
			c := newCell(x, y)
			cells[x] = append(cells[x], c)
		}
	}

	return cells
}

func newCell(x, y int) *cell {
	points := make([]float32, len(square), len(square))
	copy(points, square)

	for i := 0; i < len(points); i++ {
		var position float32
		var size float32
		switch i % 3 {
		case 0:
			size = 1.0 / float32(columns)
			position = float32(x) * size
		case 1:
			size = 1.0 / float32(rows)
			position = float32(y) * size
		default:
			continue
		}

		if points[i] < 0 {
			points[i] = (position * 2) - 1
		} else {
			points[i] = ((position + size) * 2) - 1
		}
	}

	return &cell{
		drawable: makeVao(points),

		x: x,
		y: y,
	}
}

func (c *cell) draw() {
	gl.BindVertexArray(c.drawable)
	gl.DrawArrays(gl.TRIANGLES, 0, int32(len(square)/3))
}

// initGlfw initializes glfw and returns a Window to use.
func initGlfw() *glfw.Window {
	if err := glfw.Init(); err != nil {
		panic(err)
	}
	glfw.WindowHint(glfw.Resizable, glfw.False)
	glfw.WindowHint(glfw.ContextVersionMajor, 4)
	glfw.WindowHint(glfw.ContextVersionMinor, 1)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)

	window, err := glfw.CreateWindow(width, height, "Conway's Game of Life", nil, nil)
	if err != nil {
		panic(err)
	}
	window.MakeContextCurrent()

	return window
}

// initOpenGL initializes OpenGL and returns an intiialized program.
func initOpenGL() uint32 {
	if err := gl.Init(); err != nil {
		panic(err)
	}
	version := gl.GoStr(gl.GetString(gl.VERSION))
	log.Println("OpenGL version", version)

	vertexShader, err := compileShader(vertexShaderSource, gl.VERTEX_SHADER)
	if err != nil {
		panic(err)
	}

	fragmentShader, err := compileShader(fragmentShaderSource, gl.FRAGMENT_SHADER)
	if err != nil {
		panic(err)
	}

	prog := gl.CreateProgram()
	gl.AttachShader(prog, vertexShader)
	gl.AttachShader(prog, fragmentShader)
	gl.LinkProgram(prog)
	return prog
}

// makeVao initializes and returns a vertex array from the points provided.
func makeVao(points []float32) uint32 {
	var vbo uint32
	gl.GenBuffers(1, &vbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
	gl.BufferData(gl.ARRAY_BUFFER, 4*len(points), gl.Ptr(points), gl.STATIC_DRAW)

	var vao uint32
	gl.GenVertexArrays(1, &vao)
	gl.BindVertexArray(vao)
	gl.EnableVertexAttribArray(0)
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, 0, nil)

	return vao
}

func compileShader(source string, shaderType uint32) (uint32, error) {
	shader := gl.CreateShader(shaderType)

	csources, free := gl.Strs(source)
	gl.ShaderSource(shader, 1, csources, nil)
	free()
	gl.CompileShader(shader)

	var status int32
	gl.GetShaderiv(shader, gl.COMPILE_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetShaderiv(shader, gl.INFO_LOG_LENGTH, &logLength)

		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetShaderInfoLog(shader, logLength, nil, gl.Str(log))

		return 0, fmt.Errorf("failed to compile %v: %v", source, log)
	}

	return shader, nil
}

func convertTo64(ar []float32) []float64 {
	newar := make([]float64, len(ar))
	for i, v := range ar {
		newar[i] = float64(v)
	}
	return newar
}

// So far it looks like the returns from this FFT library are
// element 0: 0Hz
// element 1: 0.5 amp * happens once
// ...
// element n/2: 1 amp * happens n/2 times
// element n/2 + 1: 0.5 amp * happens n/2 - 1 times
// ...
// element n-1: 0.5 amp * happens once
// all are cosine funcs
// magnitude is * n
func fftMag(in []float32) []float64 {
	cmplxResults := fft.FFTReal(convertTo64(in))
	toReturn := make([]float64, (len(cmplxResults)/2)-1)
	for i := 1; i < len(cmplxResults)/2; i++ {
		toReturn[i-1] = cmplx.Abs(cmplxResults[i]) + cmplx.Abs(cmplxResults[len(cmplxResults)-i])
	}
	return toReturn
}

func binFreq(in []float64, bins int) []int {
	toReturn := make([]float64, bins)
	binSize := len(in) / bins
	for i, y := range in {
		bin := i / binSize
		if bin >= bins {
			continue
		}
		toReturn[bin] += y / float64(16)
	}

	fmt.Println(toReturn)

	toReturnInts := make([]int, bins)
	for i := range toReturn {
		toReturnInts[i] = int(math.Ceil(toReturn[i]))
	}
	return toReturnInts
}

func main() {
	audioInterface, err := NewPortAudioInterface()
	if err != nil {
		fmt.Println("Error creating audio interface", err)
		return
	}

	runtime.LockOSThread()

	window := initGlfw()
	defer glfw.Terminate()
	program := initOpenGL()

	cells := makeCells()

	frameBuffer := make([]float32, 1602) // this should be 100ms worth
	for !window.ShouldClose() {
		time.Sleep(100 * time.Millisecond)
		n, err := audioInterface.ReadSamples(frameBuffer)
		if err != nil {
			fmt.Println("Read Error", err)
			return
		}

		if n < len(frameBuffer) {
			fmt.Println("Underread")
		}

		fmt.Println(n)
		//go fmt.Println()

		heights := binFreq(fftMag(frameBuffer), columns)
		draw(cells, heights, window, program)

	}
}
