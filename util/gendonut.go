package util

import (
	"image"
	"image/color"
)

func GenDonut(innerRad, outerRad int) image.Image {
	dim := image.Rectangle{image.Point{-outerRad, -outerRad}, image.Point{outerRad + 1, outerRad + 1}}
	img := image.NewAlpha(dim)
	for y := dim.Min.Y; y < dim.Max.Y; y++ {
		for x := dim.Min.X; x < dim.Max.X; x++ {
			var a uint8
			if innerRad*innerRad < x*x+y*y && x*x+y*y <= outerRad*outerRad {
				a = 255
			}
			img.SetAlpha(x, y, color.Alpha{a})
		}
	}
	return img
}
