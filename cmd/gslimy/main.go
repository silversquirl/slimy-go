package main

import (
	"log"
	"runtime"

	"github.com/go-gl/gl/v4.3-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
)

func init() {
	runtime.LockOSThread()
}

type App struct {
	worldSeed int64

	win *glfw.Window

	vao       uint32
	slimeProg uint32
	gridProg  uint32

	damaged bool
	clicked bool
	sx, sy  float64
	w, h    int32

	panX, panZ, zoom float32
}

func NewApp() (app *App, err error) {
	app = &App{worldSeed: 1, zoom: 40}
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

	gl.CreateVertexArrays(1, &app.vao)
	gl.BindVertexArray(app.vao)

	app.slimeProg, err = buildShader(fsVert, slimeFrag)
	if err != nil {
		return nil, err
	}

	app.gridProg, err = buildShader(fsVert, gridFrag)
	if err != nil {
		return nil, err
	}

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
	gl.Uniform1i64ARB(2, app.worldSeed)
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
	gl.DrawArrays(gl.TRIANGLES, 0, 3)

	gl.UseProgram(app.gridProg)
	app.SetUniforms()
	gl.DrawArrays(gl.TRIANGLES, 0, 3)

	app.win.SwapBuffers()
}

func (app *App) CursorPos(_ *glfw.Window, x, y float64) {
	if app.clicked {
		dx, dy := x-app.sx, y-app.sy
		app.panX -= float32(dx)
		app.panZ += float32(dy)
		app.sx, app.sy = x, y
		app.Damage()
	}
}
func (app *App) MouseButton(_ *glfw.Window, btn glfw.MouseButton, act glfw.Action, mods glfw.ModifierKey) {
	if btn == glfw.MouseButtonLeft {
		app.clicked = act == glfw.Press
		app.sx, app.sy = app.win.GetCursorPos()
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

func main() {
	if err := glfw.Init(); err != nil {
		log.Fatal(err)
	}
	defer glfw.Terminate()

	app, err := NewApp()
	if err != nil {
		log.Fatal(err)
	}
	defer app.Destroy()
	app.Main()
}
