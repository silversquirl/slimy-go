package gpu

import (
	"image"
	"unsafe"

	"github.com/go-gl/gl/v4.3-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/vktec/slimy"
)

type Searcher struct {
	win  *glfw.Window
	prog uint32

	MaskTex   uint32
	MaskDim   image.Point
	countBuf  uint32
	resultBuf uint32
}

func NewSearcher(shareWith *glfw.Window, mask image.Image) (s *Searcher, err error) {
	s = &Searcher{}

	if err := glfw.Init(); err != nil {
		return nil, err
	}

	glfw.WindowHint(glfw.ContextVersionMajor, 4)
	glfw.WindowHint(glfw.ContextVersionMinor, 3)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.OpenGLDebugContext, glfw.True)
	glfw.WindowHint(glfw.Visible, glfw.False)
	s.win, err = glfw.CreateWindow(1, 1, "slimy GPU search", nil, shareWith)
	if err != nil {
		return nil, err
	}

	s.activate()
	gl.DebugMessageCallback(DebugMsg, nil)

	s.prog, err = BuildComputeShader(searchComp)
	if err != nil {
		return nil, err
	}

	s.MaskDim = mask.Bounds().Canon().Size()
	gl.GenTextures(1, &s.MaskTex)
	gl.BindTexture(gl.TEXTURE_RECTANGLE, s.MaskTex)
	gl.TexParameteri(gl.TEXTURE_RECTANGLE, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_BORDER)
	gl.TexParameteri(gl.TEXTURE_RECTANGLE, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_BORDER)
	gl.TexParameterfv(gl.TEXTURE_RECTANGLE, gl.TEXTURE_BORDER_COLOR, &[]float32{0, 0, 0, 1}[0])
	UploadMask(gl.TEXTURE_RECTANGLE, mask)
	gl.BindTexture(gl.TEXTURE_RECTANGLE, 0)

	// TODO: try out other usage combinations including STREAM, DRAW and READ
	gl.GenBuffers(1, &s.countBuf)
	gl.GenBuffers(1, &s.resultBuf)
	gl.BindBuffer(gl.SHADER_STORAGE_BUFFER, s.resultBuf)
	gl.BufferData(gl.SHADER_STORAGE_BUFFER, 3*4*maxResults, nil, gl.DYNAMIC_COPY)
	gl.BindBuffer(gl.SHADER_STORAGE_BUFFER, 0)

	return s, nil
}

func (s *Searcher) Destroy() {
	gl.DeleteProgram(s.prog)
	gl.DeleteTextures(1, &s.MaskTex)
	gl.DeleteBuffers(1, &s.countBuf)
	gl.DeleteBuffers(1, &s.resultBuf)
	s.win.Destroy()
	glfw.Terminate()
}

func (s *Searcher) activate() {
	s.win.MakeContextCurrent()
	if err := gl.Init(); err != nil {
		panic(err)
	}
}

// TODO: allow more than this arbitrary limit
// >1mil results is probably fine for now tho, unless someone searches with stupidly relaxed requirements
const maxResults = 1 << 20

func (s *Searcher) Search(x0, z0, x1, z1 int32, threshold int, worldSeed int64) []slimy.Result {
	// TODO: search asynchronously or on a different thread so we don't block rendering
	// TODO: split large searches into multiple batches

	// Adjust search region so we scan all centres within the box rather than corners
	x0 -= int32(s.MaskDim.X / 2)
	x1 -= int32(s.MaskDim.X / 2)
	z0 -= int32(s.MaskDim.Y / 2)
	z1 -= int32(s.MaskDim.Y / 2)

	s.activate()
	gl.UseProgram(s.prog)
	gl.Uniform2i(0, x0, z0)
	gl.Uniform1i64ARB(1, worldSeed)
	gl.Uniform1ui(2, uint32(threshold))

	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_RECTANGLE, s.MaskTex)

	gl.BindBuffer(gl.ATOMIC_COUNTER_BUFFER, s.countBuf)
	defer gl.BindBuffer(gl.ATOMIC_COUNTER_BUFFER, 0)
	var resultCountVal uint32
	gl.BufferData(gl.ATOMIC_COUNTER_BUFFER, 4, gl.Ptr(&resultCountVal), gl.DYNAMIC_COPY)
	gl.BindBufferBase(gl.ATOMIC_COUNTER_BUFFER, 0, s.countBuf)

	gl.BindBuffer(gl.SHADER_STORAGE_BUFFER, s.resultBuf)
	defer gl.BindBuffer(gl.SHADER_STORAGE_BUFFER, 0)
	gl.BindBufferBase(gl.SHADER_STORAGE_BUFFER, 1, s.resultBuf)

	// Execute shader
	gl.DispatchComputeGroupSizeARB(uint32(x1-x0), uint32(z1-z0), 1, uint32(s.MaskDim.X), uint32(s.MaskDim.Y), 1)
	gl.MemoryBarrier(gl.ATOMIC_COUNTER_BARRIER_BIT | gl.SHADER_STORAGE_BARRIER_BIT)

	// Load results
	gl.GetBufferSubData(gl.ATOMIC_COUNTER_BUFFER, 0, 4, gl.Ptr(&resultCountVal))
	if resultCountVal > 0 {
		gpuResults := make([]gpuResult, resultCountVal)
		gl.GetBufferSubData(gl.SHADER_STORAGE_BUFFER, 0, len(gpuResults)*int(unsafe.Sizeof(gpuResults[0])), gl.Ptr(gpuResults))

		results := make([]slimy.Result, len(gpuResults))
		centerOffX, centerOffZ := int32(s.MaskDim.X/2), int32(s.MaskDim.Y/2)
		for i, gpuRes := range gpuResults {
			res := slimy.Result{
				x0 + int32(gpuRes.xoff) + centerOffX,
				z0 + int32(gpuRes.zoff) + centerOffZ,
				uint(gpuRes.count),
			}
			// TODO: use a faster sorting alg
			for i > 0 && res.OrderBefore(results[i-1]) {
				results[i] = results[i-1]
				i--
			}
			results[i] = res
		}

		return results
	} else {
		return nil
	}
}

type gpuResult struct {
	xoff, zoff, count, _ uint32
}
