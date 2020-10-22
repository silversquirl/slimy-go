package main

import "fmt"

func main() {
	mask := Mask{128 / 16, 24 / 16}
	world := World(1)
	for _, result := range world.Search(-200, -200, 200, 200, 37, mask) {
		fmt.Println(result)
	}
}
