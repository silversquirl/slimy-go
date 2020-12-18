package main

import (
	"errors"
	"flag"
	"fmt"
	"image"
	"log"
	"math"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
	"unsafe"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"github.com/go-gl/gl/v4.3-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
)

func init() {
	runtime.LockOSThread()
}

type App struct {
	worldSeed int64
	threshold int

	win *glfw.Window

	vao        uint32
	maskTex    uint32
	maskDim    image.Point
	slimeProg  uint32
	maskProg   uint32
	gridProg   uint32
	searchProg uint32

	results []Result
	damaged bool
	clicked bool
	sx, sy  float64
	w, h    int32

	panX, panZ, zoom float32
}

func NewApp(worldSeed int64, threshold int, centerPos [2]float32, maskImg image.Image, vsync bool) (app *App, err error) {
	app = &App{
		worldSeed: worldSeed,
		threshold: threshold,

		panX: centerPos[0],
		panZ: -centerPos[1],
		zoom: 40,
	}

	glfw.WindowHint(glfw.ContextVersionMajor, 4)
	glfw.WindowHint(glfw.ContextVersionMinor, 3)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.OpenGLDebugContext, glfw.True)
	app.win, err = glfw.CreateWindow(800, 600, "Slimy", nil, nil)
	if err != nil {
		return nil, err
	}

	app.win.MakeContextCurrent()
	if err = gl.Init(); err != nil {
		return nil, err
	}
	gl.DebugMessageCallback(debugMsg, nil)
	gl.ClearColor(0.1, 0.1, 0.1, 1.0)
	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
	if !vsync {
		glfw.SwapInterval(0)
	}

	gl.CreateVertexArrays(1, &app.vao)
	gl.BindVertexArray(app.vao)

	app.slimeProg, err = buildShader(fsVert, slimeFrag)
	if err != nil {
		return nil, err
	}

	app.maskProg, err = buildShader(fsVert, maskFrag)
	if err != nil {
		return nil, err
	}

	app.gridProg, err = buildShader(fsVert, gridFrag)
	if err != nil {
		return nil, err
	}

	app.searchProg, err = buildComputeShader(searchComp)
	if err != nil {
		return nil, err
	}

	app.maskDim = maskImg.Bounds().Canon().Size()
	gl.GenTextures(1, &app.maskTex)
	gl.BindTexture(gl.TEXTURE_RECTANGLE, app.maskTex)
	gl.TexParameteri(gl.TEXTURE_RECTANGLE, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_BORDER)
	gl.TexParameteri(gl.TEXTURE_RECTANGLE, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_BORDER)
	gl.TexParameterfv(gl.TEXTURE_RECTANGLE, gl.TEXTURE_BORDER_COLOR, &[]float32{0, 0, 0, 1}[0])
	uploadMask(gl.TEXTURE_RECTANGLE, maskImg)

	app.win.SetCursorPosCallback(app.CursorPos)
	app.win.SetMouseButtonCallback(app.MouseButton)
	app.win.SetScrollCallback(app.Scroll)
	app.win.SetRefreshCallback(app.Refresh)
	app.win.SetSizeCallback(app.Resize)

	app.Damage()
	return app, nil
}

func (app *App) Destroy() {
	app.win.Destroy()
}

func (app *App) Main() {
	for !app.win.ShouldClose() {
		app.Draw()
		glfw.WaitEvents()
	}
}

func (app *App) SetUniforms() {
	gl.Uniform3f(0, app.panX, app.panZ, app.zoom)
	gl.Uniform2i(1, app.w, app.h)
}

type Result struct {
	X, Z     int32
	Count, _ uint32
}

func (a Result) OrderBefore(b Result) bool {
	if a.Count != b.Count {
		return a.Count > b.Count
	} else if a.X != b.X {
		return a.X < b.X
	} else if a.Z != b.Z {
		return a.Z < b.Z
	}
	return false
}

// TODO: allow more than this arbitrary limit
// >1mil results is probably fine for now tho, unless someone searches with stupidly relaxed requirements
const maxResults = 1 << 20

func (app *App) RunSearch(x0, z0, x1, z1 int32) {
	// TODO: search asynchronously or on a different thread so we don't block rendering

	// Adjust search region so we scan all centres within the box rather than corners
	x0 -= int32(app.maskDim.X / 2)
	x1 -= int32(app.maskDim.X / 2)
	z0 -= int32(app.maskDim.Y / 2)
	z1 -= int32(app.maskDim.Y / 2)
	fmt.Printf("Searching (%d, %d) to (%d, %d)\n", x0, z0, x1, z1)

	gl.UseProgram(app.searchProg)
	gl.Uniform2i(0, x0, z0)
	gl.Uniform1i64ARB(1, app.worldSeed)
	gl.Uniform1ui(2, uint32(app.threshold))

	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_RECTANGLE, app.maskTex)

	// TODO: try out other usage combinations including STREAM, DRAW and READ
	var resultCount, results uint32
	gl.GenBuffers(1, &resultCount)
	gl.BindBuffer(gl.ATOMIC_COUNTER_BUFFER, resultCount)
	var resultCountVal uint32
	gl.BufferData(gl.ATOMIC_COUNTER_BUFFER, 4, gl.Ptr(&resultCountVal), gl.DYNAMIC_COPY)
	gl.BindBufferBase(gl.ATOMIC_COUNTER_BUFFER, 0, resultCount)

	gl.GenBuffers(1, &results)
	gl.BindBuffer(gl.SHADER_STORAGE_BUFFER, results)
	gl.BufferData(gl.SHADER_STORAGE_BUFFER, 3*4*maxResults, nil, gl.DYNAMIC_COPY)
	gl.BindBufferBase(gl.SHADER_STORAGE_BUFFER, 1, results)

	// Execute shader
	start := time.Now()
	gl.DispatchComputeGroupSizeARB(uint32(x1-x0), uint32(z1-z0), 1, uint32(app.maskDim.X), uint32(app.maskDim.Y), 1)
	gl.MemoryBarrier(gl.ATOMIC_COUNTER_BARRIER_BIT | gl.SHADER_STORAGE_BARRIER_BIT)

	// Load results
	gl.GetBufferSubData(gl.ATOMIC_COUNTER_BUFFER, 0, 4, gl.Ptr(&resultCountVal))
	end := time.Now()
	fmt.Printf("Search finished in %s\n", end.Sub(start))
	if resultCountVal > 0 {
		app.results = make([]Result, resultCountVal)
		gl.GetBufferSubData(gl.SHADER_STORAGE_BUFFER, 0, int(unsafe.Sizeof(Result{})*uintptr(resultCountVal)), gl.Ptr(app.results))

		sort.Slice(app.results, func(i, j int) bool {
			return app.results[i].OrderBefore(app.results[j])
		})

		fmt.Printf("%d results:\n", len(app.results))
		for i := range app.results {
			app.results[i].X += x0
			app.results[i].Z += z0
			fmt.Printf("  (%4d, %4d)  %d\n",
				app.results[i].X+int32(app.maskDim.X/2),
				app.results[i].Z+int32(app.maskDim.Y/2),
				app.results[i].Count)
		}
	} else {
		fmt.Println("No results")
		app.results = nil
	}
}

func (app *App) Damage() {
	app.damaged = true
}
func (app *App) Draw() {
	if !app.damaged {
		return
	}
	app.damaged = false

	gl.Clear(gl.COLOR_BUFFER_BIT)

	gl.UseProgram(app.slimeProg)
	app.SetUniforms()
	gl.Uniform1i64ARB(2, app.worldSeed)
	gl.DrawArrays(gl.TRIANGLES, 0, 3)

	gl.UseProgram(app.gridProg)
	app.SetUniforms()
	gl.DrawArrays(gl.TRIANGLES, 0, 3)

	if len(app.results) > 0 {
		gl.UseProgram(app.maskProg)
		app.SetUniforms()
		gl.ActiveTexture(gl.TEXTURE0)
		gl.BindTexture(gl.TEXTURE_RECTANGLE, app.maskTex)
		gl.Uniform2i(2, app.results[0].X, app.results[0].Z)
		gl.DrawArrays(gl.TRIANGLES, 0, 3)
	}

	app.win.SwapBuffers()
}

func ifloor(f float32) int32 {
	if math.Float32bits(f)>>31 != 0 {
		return -int32(-f)
	} else {
		return int32(f)
	}
}

// MIRROR CHANGES IN shaders.go:fcoord
func (app *App) coord(px, py int32) (chx, chz int32) {
	chx = +ifloor(float32(px-app.w/2)/app.zoom + app.panX)
	chz = -ifloor(float32(py-app.h/2)/app.zoom + app.panZ)
	return
}

func (app *App) CursorPos(_ *glfw.Window, x, y float64) {
	if app.clicked {
		dx, dy := x-app.sx, y-app.sy
		app.panX -= float32(dx) / app.zoom
		app.panZ += float32(dy) / app.zoom
		app.sx, app.sy = x, y
		app.Damage()
	}
}
func (app *App) MouseButton(_ *glfw.Window, btn glfw.MouseButton, act glfw.Action, mods glfw.ModifierKey) {
	if btn == glfw.MouseButtonLeft {
		app.clicked = act == glfw.Press
		app.sx, app.sy = app.win.GetCursorPos()
	} else if btn == glfw.MouseButtonMiddle && act == glfw.Press {
		x0, z0 := app.coord(0, app.h)
		x1, z1 := app.coord(app.w, 0)
		app.RunSearch(x0, z0, x1, z1)
		app.Damage()
	} else if btn == glfw.MouseButtonRight && act == glfw.Press {
		if len(app.results) > 0 {
			app.results = app.results[1:]
			app.Damage()
		}
	}
}
func (app *App) Scroll(_ *glfw.Window, x, y float64) {
	app.zoom += 5 * float32(y)
	if app.zoom < 5 {
		app.zoom = 5
	}
	app.Damage()
}
func (app *App) Refresh(_ *glfw.Window) {
	app.Damage()
}
func (app *App) Resize(_ *glfw.Window, w, h int) {
	gl.Viewport(0, 0, int32(w), int32(h))
	app.w, app.h = int32(w), int32(h)
	app.Damage()
}

func parsePos(s string) (pos [2]float32, err error) {
	parts := strings.Split(s, ",")
	if len(parts) != 2 {
		return [2]float32{}, errors.New("Position must be of the form 'X,Z')")
	}

	for i := 0; i < 2; i++ {
		f, err := strconv.ParseFloat(strings.Trim(parts[i], " \t\r\n"), 32)
		if err != nil {
			return [2]float32{}, err
		}
		pos[i] = float32(f)
	}
	return
}

func main() {
	seed := flag.Int64("seed", -1, "world seed")
	threshold := flag.Int("threshold", 35, "slime chunk threshold")
	mask := flag.String("mask", "", "mask image")
	pos := flag.String("pos", "0,0", "starting center position")
	vsync := flag.Bool("vsync", true, "enable vsync")
	flag.Parse()
	if *seed < 0 {
		log.Fatal("-seed must be specified")
	}

	var maskImg image.Image
	if *mask == "" {
		maskImg = genDonut(1, 8)
	} else {
		f, err := os.Open(*mask)
		if err != nil {
			log.Fatal(err)
		}
		maskImg, _, err = image.Decode(f)
		f.Close()
		if err != nil {
			log.Fatal(err)
		}
	}

	centerPos, err := parsePos(*pos)
	if err != nil {
		log.Fatal(err)
	}

	if err := glfw.Init(); err != nil {
		log.Fatal(err)
	}
	defer glfw.Terminate()

	app, err := NewApp(*seed, *threshold, centerPos, maskImg, *vsync)
	if err != nil {
		log.Fatal(err)
	}
	defer app.Destroy()
	app.Main()
}
