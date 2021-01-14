package main

import (
	"image"
	"math"
	"runtime"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/vktec/gldebug"
	"github.com/vktec/gll"
	"github.com/vktec/slimy"
	"github.com/vktec/slimy/gpu"
)

func init() {
	runtime.LockOSThread()
}

type App struct {
	gll.GL420

	worldSeed int64
	threshold int

	win *glfw.Window
	vao uint32
	s   *gpu.Searcher

	useInt64  bool
	slimeProg uint32
	maskProg  uint32
	gridProg  uint32
	maskTex   uint32
	maskDim   image.Point

	uSlimeView, uSlimeDim, uSlimeWorldSeed, uSlimeWorldSeedV int32
	uMaskView, uMaskDim, uMaskOrigin                         int32
	uGridView, uGridDim                                      int32

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
	glfw.WindowHint(glfw.ContextVersionMinor, 2)
	glfw.WindowHint(glfw.ContextCreationAPI, glfw.EGLContextAPI)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.OpenGLDebugContext, glfw.True)
	app.win, err = glfw.CreateWindow(800, 600, "Slimy", nil, nil)
	if err != nil {
		return nil, err
	}

	app.activate()
	app.DebugMessageCallbackARB(gldebug.MessageCallback)
	app.ClearColor(0.1, 0.1, 0.1, 1.0)
	app.Enable(gll.BLEND)
	app.BlendFunc(gll.SRC_ALPHA, gll.ONE_MINUS_SRC_ALPHA)
	if !vsync {
		glfw.SwapInterval(0)
	}

	app.GenVertexArrays(1, &app.vao)
	app.BindVertexArray(app.vao)

	app.useInt64 = gpu.ExtensionSupported(app, "GL_ARB_gpu_shader_int64")
	app.slimeProg, err = gpu.BuildShader(app, fsVert, slimeFrag)
	if err != nil {
		return nil, err
	}
	app.uSlimeView = app.GetUniformLocation(app.slimeProg, gll.Str("view\000"))
	app.uSlimeDim = app.GetUniformLocation(app.slimeProg, gll.Str("dim\000"))
	app.uSlimeWorldSeed = app.GetUniformLocation(app.slimeProg, gll.Str("worldSeed\000"))
	app.uSlimeWorldSeedV = app.GetUniformLocation(app.slimeProg, gll.Str("worldSeedV\000"))

	app.maskProg, err = gpu.BuildShader(app, fsVert, maskFrag)
	if err != nil {
		return nil, err
	}
	app.uMaskView = app.GetUniformLocation(app.maskProg, gll.Str("view\000"))
	app.uMaskDim = app.GetUniformLocation(app.maskProg, gll.Str("dim\000"))
	app.uMaskOrigin = app.GetUniformLocation(app.maskProg, gll.Str("origin\000"))

	app.gridProg, err = gpu.BuildShader(app, fsVert, gridFrag)
	if err != nil {
		return nil, err
	}
	app.uGridView = app.GetUniformLocation(app.gridProg, gll.Str("view\000"))
	app.uGridDim = app.GetUniformLocation(app.gridProg, gll.Str("dim\000"))

	app.maskDim = maskImg.Bounds().Canon().Size()
	app.maskTex = gpu.UploadMask(app, maskImg)

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
	app.GL420 = gll.New420(glfw.GetProcAddress)
}

func (app *App) Main() {
	for !app.win.ShouldClose() {
		app.Draw()
		glfw.WaitEvents()
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

	app.activate()
	app.Clear(gll.COLOR_BUFFER_BIT)

	app.UseProgram(app.slimeProg)
	app.Uniform3f(app.uSlimeView, app.panX, app.panZ, app.zoom)
	app.Uniform2i(app.uSlimeDim, app.w, app.h)
	if app.useInt64 {
		app.Uniform1i64ARB(app.uSlimeWorldSeed, app.worldSeed)
	}
	app.Uniform2ui(app.uSlimeWorldSeedV, uint32(app.worldSeed>>32), uint32(app.worldSeed))
	app.DrawArrays(gll.TRIANGLES, 0, 3)

	app.UseProgram(app.gridProg)
	app.Uniform3f(app.uGridView, app.panX, app.panZ, app.zoom)
	app.Uniform2i(app.uGridDim, app.w, app.h)
	app.DrawArrays(gll.TRIANGLES, 0, 3)

	if len(app.results) > 0 {
		app.UseProgram(app.maskProg)
		app.Uniform3f(app.uMaskView, app.panX, app.panZ, app.zoom)
		app.Uniform2i(app.uMaskDim, app.w, app.h)

		px := app.results[0].X - int32(app.maskDim.X/2)
		pz := app.results[0].Z - int32(app.maskDim.Y/2)
		app.Uniform2i(app.uMaskOrigin, px, pz)

		app.ActiveTexture(gll.TEXTURE0)
		app.BindTexture(gll.TEXTURE_RECTANGLE, app.maskTex)
		app.DrawArrays(gll.TRIANGLES, 0, 3)
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
	app.Viewport(0, 0, int32(w), int32(h))
	app.w, app.h = int32(w), int32(h)
	app.Damage()
}
