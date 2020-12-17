package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"strconv"

	"github.com/vktec/slimy"
)

func formatCSV(results []slimy.SearchResult) error {
	if _, err := fmt.Println("Center Chunk X,Center Chunk Z,Slime Chunk Count"); err != nil {
		return err
	}
	for _, result := range results {
		if _, err := fmt.Print(result.X, ",", result.Z, ",", result.Count, "\n"); err != nil {
			return err
		}
	}
	return nil
}

func formatJSON(results []slimy.SearchResult) error {
	return json.NewEncoder(os.Stdout).Encode(results)
}

func formatHuman(results []slimy.SearchResult) error {
	for _, result := range results {
		if _, err := fmt.Printf("(%6d, %6d) %3d chunks\n", result.X, result.Z, result.Count); err != nil {
			return err
		}
	}
	return nil
}

func main() {
	workerCount := flag.Int("j", runtime.GOMAXPROCS(0), "Number of concurrent workers")
	outputFormat := flag.String("f", "human", "Output `format` (valid options: csv, json, human)")
	drawFile := flag.String("draw", "", "Output a PNG `file`")

	flag.CommandLine.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [-j count] [-f format] range seed threshold\n\n", filepath.Base(os.Args[0]))
		flag.PrintDefaults()
		fmt.Fprintln(os.Stderr)
	}
	flag.Parse()

	if flag.NArg() < 3 {
		flag.CommandLine.Usage()
		os.Exit(1)
	}

	searchRange64, err := strconv.ParseInt(flag.Arg(0), 10, 32)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Could not convert range to integer:", err)
		os.Exit(2)
	}
	searchRange := int32(searchRange64)
	if searchRange < 0 {
		fmt.Fprintln(os.Stderr, "Range must not be negative")
		os.Exit(2)
	}

	// TODO: textual seeds
	seed, err := strconv.ParseInt(flag.Arg(1), 10, 64)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Could not convert seed to integer:", err)
		os.Exit(2)
	}

	threshold64, err := strconv.ParseInt(flag.Arg(2), 10, 0)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Could not convert threshold to integer:", err)
		os.Exit(2)
	}
	threshold := int(threshold64)
	// TODO: use negative threshold for max count
	if threshold < 0 {
		fmt.Fprintln(os.Stderr, "Threshold must not be negative")
		os.Exit(2)
	}

	var fmter func([]slimy.SearchResult) error
	switch *outputFormat {
	case "csv":
		fmter = formatCSV
	case "json":
		fmter = formatJSON
	case "human":
		fmter = formatHuman
	default:
		fmt.Fprintln(os.Stderr, "Format must be one of: csv, json, human")
		os.Exit(2)
	}

	mask := slimy.Mask{ORad: 8, IRad: 1}
	world := slimy.World(seed)
	if *drawFile != "" {
		// TODO: don't require threshold
		img := image.NewRGBA(image.Rectangle{
			image.Point{int(-searchRange), int(-searchRange)},
			image.Point{int(searchRange), int(searchRange)},
		})
		world.DrawArea(*workerCount, img)
		f, err := os.Create(*drawFile)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error opening image file:", err)
			os.Exit(2)
		}
		err = png.Encode(f, img)
		f.Close()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error writing image:", err)
			os.Exit(2)
		}
	} else {
		fmter(world.Search(*workerCount, -searchRange, -searchRange, searchRange, searchRange, threshold, mask))
	}
}
