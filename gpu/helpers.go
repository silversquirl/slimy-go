package gpu

import (
	"image"

	"github.com/vktec/gll"
	"github.com/vktec/gll/glh"
)

func BuildShader(gl gll.GL330, vert, frag string) (prog uint32, err error) {
	vshad, err := glh.NewShader(gl, gll.VERTEX_SHADER, vert)
	if err != nil {
		return 0, err
	}
	fshad, err := glh.NewShader(gl, gll.FRAGMENT_SHADER, frag)
	if err != nil {
		return 0, err
	}
	return glh.NewProgram(gl, vshad, fshad)
}
func BuildComputeShader(gl gll.GL430, source string) (prog uint32, err error) {
	shad, err := glh.NewShader(gl, gll.COMPUTE_SHADER, source)
	if err != nil {
		return 0, err
	}
	return glh.NewProgram(gl, shad)
}

func UploadMask(gl gll.GL330, img image.Image) (tex uint32) {
	gl.GenTextures(1, &tex)
	gl.BindTexture(gll.TEXTURE_RECTANGLE, tex)
	gl.TexParameteri(gll.TEXTURE_RECTANGLE, gll.TEXTURE_WRAP_S, gll.CLAMP_TO_BORDER)
	gl.TexParameteri(gll.TEXTURE_RECTANGLE, gll.TEXTURE_WRAP_T, gll.CLAMP_TO_BORDER)
	gl.TexParameterfv(gll.TEXTURE_RECTANGLE, gll.TEXTURE_BORDER_COLOR, &[]float32{0, 0, 0, 1}[0])

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
	gl.TexImage2D(gll.TEXTURE_RECTANGLE, 0, gll.R8, int32(dim.Dx()), int32(dim.Dy()), 0, gll.RGBA, gll.UNSIGNED_BYTE, gll.Ptr(data))

	gl.BindTexture(gll.TEXTURE_RECTANGLE, 0)
	return tex
}
