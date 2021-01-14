package gpu

import (
	"image"
	"unsafe"

	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/vktec/gldebug"
	"github.com/vktec/glhl"
	"github.com/vktec/gll"
	"github.com/vktec/slimy"
)

type Searcher struct {
	gll.GL430
	ctx  glhl.Context
	prog uint32

	useInt64  bool
	maskTex   uint32
	maskDim   image.Point
	countBuf  uint32
	resultBuf uint32

	uOffset, uThreshold, uWorldSeed, uWorldSeedV int32
}

func NewSearcher(mask image.Image) (*Searcher, error) {
	ctx, err := glhl.NewContext(4, 3, glhl.Core|glhl.Debug)

	s := &Searcher{ctx: ctx}
	s.activate()
	s.DebugMessageCallback(gldebug.MessageCallback)

	s.useInt64 = ExtensionSupported(s, "GL_ARB_gpu_shader_int64")
	s.prog, err = BuildComputeShader(s, searchComp)
	if err != nil {
		return nil, err
	}
	s.uOffset = s.GetUniformLocation(s.prog, gll.Str("offset\000"))
	s.uThreshold = s.GetUniformLocation(s.prog, gll.Str("threshold\000"))
	s.uWorldSeed = s.GetUniformLocation(s.prog, gll.Str("worldSeed\000"))
	s.uWorldSeedV = s.GetUniformLocation(s.prog, gll.Str("worldSeedV\000"))

	s.maskDim = mask.Bounds().Canon().Size()
	s.maskTex = UploadMask(s, mask)

	// TODO: try out other usage combinations including STREAM, DRAW and READ
	s.GenBuffers(1, &s.countBuf)
	s.GenBuffers(1, &s.resultBuf)
	s.BindBuffer(gll.SHADER_STORAGE_BUFFER, s.resultBuf)
	s.BufferData(gll.SHADER_STORAGE_BUFFER, 3*4*maxResults, nil, gll.DYNAMIC_COPY)
	s.BindBuffer(gll.SHADER_STORAGE_BUFFER, 0)

	return s, nil
}

func (s *Searcher) Destroy() {
	s.DeleteProgram(s.prog)
	s.DeleteTextures(1, &s.maskTex)
	s.DeleteBuffers(1, &s.countBuf)
	s.DeleteBuffers(1, &s.resultBuf)
	s.ctx.Destroy()
	glfw.Terminate()
}

func (s *Searcher) activate() {
	s.ctx.MakeContextCurrent()
	s.GL430 = gll.New430(glhl.GetProcAddr)
}

// TODO: allow more than this arbitrary limit
// >1mil results is probably fine for now tho, unless someone searches with stupidly relaxed requirements
const maxResults = 1 << 20

func (s *Searcher) Search(x0, z0, x1, z1 int32, threshold int, worldSeed int64) []slimy.Result {
	// TODO: search asynchronously or on a different thread so we don't block rendering
	// TODO: split large searches into multiple batches

	// Adjust search region so we scan all centres within the box rather than corners
	x0 -= int32(s.maskDim.X / 2)
	x1 -= int32(s.maskDim.X / 2)
	z0 -= int32(s.maskDim.Y / 2)
	z1 -= int32(s.maskDim.Y / 2)

	s.activate()
	s.UseProgram(s.prog)
	s.Uniform2i(s.uOffset, x0, z0)
	s.Uniform1ui(s.uThreshold, uint32(threshold))
	if s.useInt64 {
		s.Uniform1i64ARB(s.uWorldSeed, worldSeed)
	}
	s.Uniform2ui(s.uWorldSeedV, uint32(worldSeed>>32), uint32(worldSeed))

	s.ActiveTexture(gll.TEXTURE0)
	s.BindTexture(gll.TEXTURE_RECTANGLE, s.maskTex)

	s.BindBuffer(gll.ATOMIC_COUNTER_BUFFER, s.countBuf)
	defer s.BindBuffer(gll.ATOMIC_COUNTER_BUFFER, 0)
	var resultCountVal uint32
	s.BufferData(gll.ATOMIC_COUNTER_BUFFER, 4, gll.Ptr(&resultCountVal), gll.DYNAMIC_COPY)
	s.BindBufferBase(gll.ATOMIC_COUNTER_BUFFER, 0, s.countBuf)

	s.BindBuffer(gll.SHADER_STORAGE_BUFFER, s.resultBuf)
	defer s.BindBuffer(gll.SHADER_STORAGE_BUFFER, 0)
	s.BindBufferBase(gll.SHADER_STORAGE_BUFFER, 1, s.resultBuf)

	// Execute shader
	s.DispatchComputeGroupSizeARB(uint32(x1-x0), uint32(z1-z0), 1, uint32(s.maskDim.X), uint32(s.maskDim.Y), 1)
	s.MemoryBarrier(gll.ATOMIC_COUNTER_BARRIER_BIT | gll.SHADER_STORAGE_BARRIER_BIT)

	// Load results
	s.GetBufferSubData(gll.ATOMIC_COUNTER_BUFFER, 0, 4, gll.Ptr(&resultCountVal))
	if resultCountVal > 0 {
		gpuResults := make([]gpuResult, resultCountVal)
		s.GetBufferSubData(gll.SHADER_STORAGE_BUFFER, 0, len(gpuResults)*int(unsafe.Sizeof(gpuResults[0])), gll.Ptr(gpuResults))

		results := make([]slimy.Result, len(gpuResults))
		centerOffX, centerOffZ := int32(s.maskDim.X/2), int32(s.maskDim.Y/2)
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
