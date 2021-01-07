package main

import (
	"image"
	"math"
	"runtime"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"github.com/go-gl/gl/v4.3-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/vktec/slimy"
	"github.com/vktec/slimy/gpu"
)

func init() {
	runtime.LockOSThread()
}

type App struct {
	worldSeed int64
	threshold int

	win *glfw.Window
	vao uint32
	s   *gpu.Searcher

	slimeProg uint32
	maskProg  uint32
	gridProg  uint32
	maskTex   uint32
	maskDim   image.Point

	results []slimy.Result
	damaged bool
	clicked bool
	sx, sy  float64
	w, h    int32

	panX, panZ, zoom float32
}

func NewApp(worldSeed int64, threshold int, centerPos [2]int, maskImg image.Image, vsync bool) (app *App, err error) {
	app = &App{
		worldSeed: worldSeed,
		threshold: threshold,

		panX: float32(centerPos[0]),
		panZ: -float32(centerPos[1]),
		zoom: 40,
	}

	if err := glfw.Init(); err != nil {
		return nil, err
	}

	glfw.WindowHint(glfw.ContextVersionMajor, 4)
	glfw.WindowHint(glfw.ContextVersionMinor, 3)
	glfw.WindowHint(glfw.ContextCreationAPI, glfw.EGLContextAPI)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.OpenGLDebugContext, glfw.True)
	app.win, err = glfw.CreateWindow(800, 600, "Slimy", nil, nil)
	if err != nil {
		return nil, err
	}

	app.activate()
	gl.DebugMessageCallback(gpu.DebugMsg, nil)
	gl.ClearColor(0.1, 0.1, 0.1, 1.0)
	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
	if !vsync {
		glfw.SwapInterval(0)
	}

	gl.CreateVertexArrays(1, &app.vao)
	gl.BindVertexArray(app.vao)

	app.slimeProg, err = gpu.BuildShader(fsVert, slimeFrag)
	if err != nil {
		return nil, err
	}

	app.maskProg, err = gpu.BuildShader(fsVert, maskFrag)
	if err != nil {
		return nil, err
	}

	app.gridProg, err = gpu.BuildShader(fsVert, gridFrag)
	if err != nil {
		return nil, err
	}

	app.maskDim = maskImg.Bounds().Canon().Size()
	app.maskTex = gpu.UploadMask(maskImg)

	app.win.SetCursorPosCallback(app.CursorPos)
	app.win.SetMouseButtonCallback(app.MouseButton)
	app.win.SetScrollCallback(app.Scroll)
	app.win.SetRefreshCallback(app.Refresh)
	app.win.SetSizeCallback(app.Resize)

	app.s, err = gpu.NewSearcher(maskImg)
	if err != nil {
		return nil, err
	}

	app.Damage()
	return app, nil
}

func (app *App) Destroy() {
	app.win.Destroy()
	glfw.Terminate()
}

func (app *App) activate() {
	app.win.MakeContextCurrent()
	if err := gl.Init(); err != nil {
		panic(err)
	}
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

func (app *App) Damage() {
	app.damaged = true
}
func (app *App) Draw() {
	if !app.damaged {
		return
	}
	app.damaged = false

	app.activate()
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
		px := app.results[0].X - int32(app.maskDim.X/2)
		pz := app.results[0].Z - int32(app.maskDim.Y/2)
		gl.Uniform2i(2, px, pz)
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

// MIRROR CHANGES IN gpu/shaders.go:Fcoord
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
		app.results = runSearch(app.s, x0, z0, x1, z1, app.threshold, app.worldSeed)
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