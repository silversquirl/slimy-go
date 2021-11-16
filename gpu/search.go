package gpu

import (
	"errors"
	"fmt"
	"image"
	"unsafe"

	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/vktec/gldebug"
	"github.com/vktec/gll"
	"github.com/vktec/slimy"
)

type Searcher struct {
	gll.GL430

	ctx interface {
		MakeContextCurrent()
		Destroy()
	}
	getProcAddr func(name string) unsafe.Pointer

	useInt64     bool
	useGroupSize bool

	prog      uint32
	maskTex   uint32
	maskDim   image.Point
	countBuf  uint32
	resultBuf uint32

	uOffset, uThreshold, uWorldSeed, uWorldSeedV int32
}

func NewGLFWSearcher(mask image.Image) (*Searcher, error) {
	if err := glfw.Init(); err != nil {
		return nil, err
	}
	glfw.WindowHint(glfw.ContextVersionMajor, 4)
	glfw.WindowHint(glfw.ContextVersionMinor, 2)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	if Debug {
		glfw.WindowHint(glfw.OpenGLDebugContext, glfw.True)
	}
	glfw.WindowHint(glfw.Visible, glfw.False)
	win, err := glfw.CreateWindow(1, 1, "slimy search context", nil, nil)
	if err != nil {
		return nil, err
	}
	s := &Searcher{ctx: win, getProcAddr: glfw.GetProcAddress}
	if err := s.init(mask); err != nil {
		s.Destroy()
		return nil, err
	}
	return s, nil
}

func (s *Searcher) init(mask image.Image) (err error) {
	s.activate()
	if Debug {
		s.DebugMessageCallback(gldebug.MessageCallback)
	}

	if !ExtensionSupported(s, "GL_ARB_compute_shader") {
		return errors.New("GL_ARB_compute_shader not supported - cannot use GPU search")
	}
	if !ExtensionSupported(s, "GL_ARB_shader_storage_buffer_object") {
		return errors.New("GL_ARB_shader_storage_buffer_object not supported - cannot use GPU search")
	}

	s.useInt64 = ExtensionSupported(s, "GL_ARB_gpu_shader_int64")
	s.useGroupSize = ExtensionSupported(s, "GL_ARB_compute_variable_group_size")

	s.maskDim = mask.Bounds().Canon().Size()
	s.maskTex = UploadMask(s, mask)

	// TODO: try out other usage combinations including STREAM, DRAW and READ
	s.GenBuffers(1, &s.countBuf)
	s.GenBuffers(1, &s.resultBuf)
	s.BindBuffer(gll.SHADER_STORAGE_BUFFER, s.resultBuf)
	s.BufferData(gll.SHADER_STORAGE_BUFFER, 4*4*resultBufferLength, nil, gll.DYNAMIC_COPY)
	s.BindBuffer(gll.SHADER_STORAGE_BUFFER, 0)

	return nil
}

func (s *Searcher) Destroy() {
	if s.prog != 0 {
		s.DeleteProgram(s.prog)
	}
	s.DeleteTextures(1, &s.maskTex)
	s.DeleteBuffers(1, &s.countBuf)
	s.DeleteBuffers(1, &s.resultBuf)
	s.ctx.Destroy()
	glfw.Terminate()
}

func (s *Searcher) activate() {
	s.ctx.MakeContextCurrent()
	s.GL430 = gll.New430(s.getProcAddr)
}

func (s *Searcher) initProg() (err error) {
	var prelude string
	if s.useGroupSize {
		if s.prog != 0 {
			return nil
		}
		prelude = searchPreludeVariable
	} else {
		if s.prog != 0 {
			s.DeleteProgram(s.prog)
		}
		prelude = fmt.Sprintf(searchPreludeFixed, s.maskDim.X, s.maskDim.Y)
	}

	s.prog, err = BuildComputeShader(s, prelude, searchComp)
	if err != nil {
		return err
	}

	s.uOffset = s.GetUniformLocation(s.prog, gll.Str("offset\000"))
	s.uThreshold = s.GetUniformLocation(s.prog, gll.Str("threshold\000"))
	s.uWorldSeed = s.GetUniformLocation(s.prog, gll.Str("worldSeed\000"))
	s.uWorldSeedV = s.GetUniformLocation(s.prog, gll.Str("worldSeedV\000"))

	return nil
}

const (
	searchRegionWidth  = 1024 // Partition the search into squares of this size
	resultBufferLength = searchRegionWidth * searchRegionWidth
)

func (s *Searcher) Search(x0, z0, x1, z1 int32, threshold int, worldSeed int64) []slimy.Result {
	// TODO: search asynchronously or on a different thread so we don't block rendering
	// TODO: split large searches into multiple batches

	// Adjust search region so we scan all centres within the box rather than corners
	x0 -= int32(s.maskDim.X / 2)
	x1 -= int32(s.maskDim.X / 2)
	z0 -= int32(s.maskDim.Y / 2)
	z1 -= int32(s.maskDim.Y / 2)

	s.activate()
	s.initProg()
	s.UseProgram(s.prog)
	s.Uniform1i(s.uThreshold, int32(threshold))
	if s.useInt64 {
		s.Uniform1i64ARB(s.uWorldSeed, worldSeed)
	}
	s.Uniform2ui(s.uWorldSeedV, uint32(worldSeed>>32), uint32(worldSeed))

	s.ActiveTexture(gll.TEXTURE0)
	s.BindTexture(gll.TEXTURE_RECTANGLE, s.maskTex)

	s.BindBuffer(gll.ATOMIC_COUNTER_BUFFER, s.countBuf)
	defer s.BindBuffer(gll.ATOMIC_COUNTER_BUFFER, 0)
	s.BindBufferBase(gll.ATOMIC_COUNTER_BUFFER, 0, s.countBuf)

	s.BindBuffer(gll.SHADER_STORAGE_BUFFER, s.resultBuf)
	defer s.BindBuffer(gll.SHADER_STORAGE_BUFFER, 0)
	s.BindBufferBase(gll.SHADER_STORAGE_BUFFER, 1, s.resultBuf)

	resultC := make(chan []slimy.Result)
	returnC := make(chan []slimy.Result)
	go func() {
		var results []slimy.Result
		for group := range resultC {
			for _, res := range group {
				i := len(results)
				results = append(results, res)
				// TODO: use a faster sorting alg
				for i > 0 && res.OrderBefore(results[i-1], threshold) {
					results[i] = results[i-1]
					i--
				}
				results[i] = res
			}
		}
		returnC <- results
	}()

	rw, rh := (x1-x0)/searchRegionWidth, (z1-z0)/searchRegionWidth
	for rz := int32(0); rz < rh; rz++ {
		rz0 := z0 + rz*searchRegionWidth
		rz1 := rz0 + searchRegionWidth
		for rx := int32(0); rx < rw; rx++ {
			rx0 := x0 + rx*searchRegionWidth
			rx1 := rx0 + searchRegionWidth
			resultC <- s.executeSearch(rx0, rz0, rx1, rz1)
		}
		if rw*searchRegionWidth < x1-x0 {
			rx0 := x0 + rw*searchRegionWidth
			resultC <- s.executeSearch(rx0, rz0, x1, rz1)
		}
	}
	if rh*searchRegionWidth < z1-z0 {
		rz0 := z0 + rh*searchRegionWidth
		for rx := int32(0); rx < rw; rx++ {
			rx0 := x0 + rx*searchRegionWidth
			rx1 := rx0 + searchRegionWidth
			resultC <- s.executeSearch(rx0, rz0, rx1, z1)
		}
		if rw*searchRegionWidth < x1-x0 {
			rx0 := x0 + rw*searchRegionWidth
			resultC <- s.executeSearch(rx0, rz0, x1, z1)
		}
	}

	close(resultC)
	return <-returnC
}

func (s *Searcher) executeSearch(x0, z0, x1, z1 int32) []slimy.Result {
	s.Uniform2i(s.uOffset, x0, z0)
	var resultCount uint32
	s.BufferData(gll.ATOMIC_COUNTER_BUFFER, 4, gll.Ptr(&resultCount), gll.DYNAMIC_COPY)

	// Execute shader
	if s.useGroupSize {
		s.DispatchComputeGroupSizeARB(uint32(x1-x0), uint32(z1-z0), 1, uint32(s.maskDim.X), uint32(s.maskDim.Y), 1)
	} else {
		s.DispatchCompute(uint32(x1-x0), uint32(z1-z0), 1)
	}
	s.MemoryBarrier(gll.ATOMIC_COUNTER_BARRIER_BIT | gll.SHADER_STORAGE_BARRIER_BIT)

	// Load results
	s.GetBufferSubData(gll.ATOMIC_COUNTER_BUFFER, 0, 4, gll.Ptr(&resultCount))
	if resultCount > 0 {
		gpuResults := make([]gpuResult, resultCount)
		// TODO: compare performance with using image load store and PBOs instead
		s.GetBufferSubData(gll.SHADER_STORAGE_BUFFER, 0, len(gpuResults)*int(unsafe.Sizeof(gpuResults[0])), gll.Ptr(gpuResults))

		results := make([]slimy.Result, resultCount)
		centerOffX, centerOffZ := int32(s.maskDim.X/2), int32(s.maskDim.Y/2)
		for i, gpuRes := range gpuResults {
			results[i] = slimy.Result{
				x0 + int32(gpuRes.xoff) + centerOffX,
				z0 + int32(gpuRes.zoff) + centerOffZ,
				uint(gpuRes.count),
			}
		}
		return results
	} else {
		return nil
	}
}

type gpuResult struct {
	xoff, zoff, count, _ uint32
}
