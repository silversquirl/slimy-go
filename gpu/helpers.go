package gpu

import (
	"errors"
	"fmt"
	"image"
	"log"
	"strings"
	"unsafe"

	"github.com/go-gl/gl/v4.3-core/gl"
)

func GetShaderError(thing uint32, ivFunc func(thing, pname uint32, params *int32), logFunc func(thing uint32, bufSize int32, length *int32, infoLog *uint8)) error {
	var bufSize int32
	ivFunc(thing, gl.INFO_LOG_LENGTH, &bufSize)

	if bufSize > 0 {
		errBuf := make([]byte, bufSize)
		var length int32
		logFunc(thing, bufSize, &length, &errBuf[0])

		errMsg := string(errBuf[:length])
		errMsg = strings.TrimRight(errMsg, "\r\n")
		return errors.New(errMsg)
	} else {
		return errors.New("No error message")
	}
}
func CompileShader(shad uint32, source string) error {
	csrc, free := gl.Strs(source)
	clen := int32(len(source))
	gl.ShaderSource(shad, 1, csrc, &clen)
	gl.CompileShader(shad)
	free()

	var result int32
	gl.GetShaderiv(shad, gl.COMPILE_STATUS, &result)
	if result == 0 {
		defer gl.DeleteShader(shad)
		return GetShaderError(shad, gl.GetShaderiv, gl.GetShaderInfoLog)
	}
	return nil
}
func BuildShader(vert, frag string) (prog uint32, err error) {
	vshad := gl.CreateShader(gl.VERTEX_SHADER)
	if err := CompileShader(vshad, vert); err != nil {
		return 0, err
	}
	fshad := gl.CreateShader(gl.FRAGMENT_SHADER)
	if err := CompileShader(fshad, frag); err != nil {
		return 0, err
	}

	prog = gl.CreateProgram()
	gl.AttachShader(prog, vshad)
	gl.AttachShader(prog, fshad)
	gl.LinkProgram(prog)
	gl.DetachShader(prog, vshad)
	gl.DetachShader(prog, fshad)

	var result int32
	gl.GetProgramiv(prog, gl.LINK_STATUS, &result)
	if result == 0 {
		defer gl.DeleteProgram(prog)
		return 0, GetShaderError(prog, gl.GetProgramiv, gl.GetProgramInfoLog)
	}
	return prog, nil
}
func BuildComputeShader(source string) (prog uint32, err error) {
	shad := gl.CreateShader(gl.COMPUTE_SHADER)
	if err := CompileShader(shad, source); err != nil {
		return 0, err
	}

	prog = gl.CreateProgram()
	gl.AttachShader(prog, shad)
	gl.LinkProgram(prog)
	gl.DetachShader(prog, shad)

	var result int32
	gl.GetProgramiv(prog, gl.LINK_STATUS, &result)
	if result == 0 {
		defer gl.DeleteProgram(prog)
		return 0, GetShaderError(prog, gl.GetProgramiv, gl.GetProgramInfoLog)
	}
	return prog, nil
}

func UploadMask(target uint32, img image.Image) {
	dim := img.Bounds().Canon()
	data := make([][4]uint8, dim.Dx()*dim.Dy())
	for y := dim.Min.Y; y < dim.Max.Y; y++ {
		for x := dim.Min.X; x < dim.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			if (r > 0x7fff || g > 0x7fff || b > 0x7fff) && a > 0x7fff {
				tx := x - dim.Min.X
				ty := y - dim.Min.Y
				data[ty*dim.Dx()+tx][0] = 0xff
			}
		}
	}
	gl.TexImage2D(target, 0, gl.R8, int32(dim.Dx()), int32(dim.Dy()), 0, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(data))
}

func DebugMsg(source, gltype, id, severity uint32, length int32, message string, userParam unsafe.Pointer) {
	var sourceStr, typeStr, severityStr string

	switch source {
	case gl.DEBUG_SOURCE_API:
		sourceStr = "API"
	case gl.DEBUG_SOURCE_WINDOW_SYSTEM:
		sourceStr = "Window System"
	case gl.DEBUG_SOURCE_SHADER_COMPILER:
		sourceStr = "Shader Compiler"
	case gl.DEBUG_SOURCE_THIRD_PARTY:
		sourceStr = "Third Party"
	case gl.DEBUG_SOURCE_APPLICATION:
		sourceStr = "Application"
	case gl.DEBUG_SOURCE_OTHER:
		sourceStr = "Other"
	}

	switch gltype {
	case gl.DEBUG_TYPE_ERROR:
		typeStr = "Error"
	case gl.DEBUG_TYPE_DEPRECATED_BEHAVIOR:
		typeStr = "Deprecated Behaviour"
	case gl.DEBUG_TYPE_UNDEFINED_BEHAVIOR:
		typeStr = "Undefined Behaviour"
	case gl.DEBUG_TYPE_PORTABILITY:
		typeStr = "Portability"
	case gl.DEBUG_TYPE_PERFORMANCE:
		typeStr = "Performance"
	case gl.DEBUG_TYPE_MARKER:
		typeStr = "Marker"
	case gl.DEBUG_TYPE_PUSH_GROUP:
		typeStr = "Push Group"
	case gl.DEBUG_TYPE_POP_GROUP:
		typeStr = "Pop Group"
	case gl.DEBUG_TYPE_OTHER:
		typeStr = "Other"
	}

	switch severity {
	case gl.DEBUG_SEVERITY_HIGH:
		severityStr = "High"
	case gl.DEBUG_SEVERITY_MEDIUM:
		severityStr = "Medium"
	case gl.DEBUG_SEVERITY_LOW:
		severityStr = "Low"
	case gl.DEBUG_SEVERITY_NOTIFICATION:
		severityStr = "Notification"
	}

	msg := fmt.Sprintf("(%d) source: %s, type: %s, severity: %s, message: %s", id, sourceStr, typeStr, severityStr, message)
	if severity == gl.DEBUG_SEVERITY_HIGH {
		panic(msg)
	} else {
		log.Println(msg)
	}
}
