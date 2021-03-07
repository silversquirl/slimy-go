package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"image"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/vktec/slimy"
	"github.com/vktec/slimy/cpu"
	"github.com/vktec/slimy/gpu"
	"github.com/vktec/slimy/util"
)

func formatCSV(results []slimy.Result) error {
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

func formatJSON(results []slimy.Result) error {
	return json.NewEncoder(os.Stdout).Encode(results)
}

func formatHuman(results []slimy.Result) error {
	if len(results) > 0 {
		if len(results) == 1 {
			if _, err := fmt.Println("1 result:"); err != nil {
				return err
			}
		} else {
			if _, err := fmt.Println(len(results), "results:"); err != nil {
				return err
			}
		}
		for _, result := range results {
			if _, err := fmt.Printf("(%6d, %6d) %3d chunks\n", result.X, result.Z, result.Count); err != nil {
				return err
			}
		}
	} else {
		if _, err := fmt.Println("No results"); err != nil {
			return err
		}
	}
	return nil
}

var fmter func([]slimy.Result) error

func runSearch(s slimy.Searcher, x0, z0, x1, z1 int32, threshold int, worldSeed int64) (results []slimy.Result) {
	fmt.Fprintf(os.Stderr, "Searching (%d, %d) to (%d, %d)\n", x0, z0, x1, z1)
	start := time.Now()
	results = s.Search(x0, z0, x1, z1, threshold, worldSeed)
	end := time.Now()
	fmt.Fprintf(os.Stderr, "Search finished in %s\n", end.Sub(start))
	fmter(results)
	return
}

func parsePos(s string) (pos [2]int, err error) {
	parts := strings.Split(s, ",")
	if len(parts) != 2 {
		return [2]int{}, errors.New("Position must be of the form 'X,Z')")
	}

	for i := 0; i < 2; i++ {
		pos[i], err = strconv.Atoi(strings.Trim(parts[i], " \t\r\n"))
		if err != nil {
			return [2]int{}, err
		}
	}
	return
}

func main() {
	workerCount := flag.Int("j", runtime.GOMAXPROCS(0), "number of concurrent workers (cpu only)")
	outputFormat := flag.String("f", "human", "output `format` (valid options: csv, json, human)")
	method := flag.String("m", "gpu", "search method to use (search mode only) (options: cpu, gpu)")
	mask := flag.String("mask", "", "mask image `file`name")
	pos := flag.String("pos", "0,0", "search center `position`")
	vsync := flag.Bool("vsync", true, "enable vsync (gui mode only)")

	flag.CommandLine.Usage = func() {
		cmd := filepath.Base(os.Args[0])
		fmt.Fprintf(os.Stderr, "Usage: %s [options] seed range threshold\n", cmd)
		fmt.Fprintf(os.Stderr, "       %s [options] seed threshold\n\n", cmd)
		flag.PrintDefaults()
		fmt.Fprintln(os.Stderr)
	}
	flag.Parse()

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

	var maskImg image.Image
	if *mask == "" {
		maskImg = util.GenDonut(1, 8)
	} else {
		f, err := os.Open(*mask)
		if err != nil {
			log.Fatal(err)
		}
		maskImg, _, err = image.Decode(f)
		f.Close()
		if err != nil {
			log.Fatal(err)
		}
	}

	centerPos, err := parsePos(*pos)
	if err != nil {
		log.Fatal(err)
	}

	switch flag.NArg() {
	default:
		flag.CommandLine.Usage()
		os.Exit(1)

	case 3:
		// Search mode
		var searcher slimy.Searcher
		switch *method {
		case "gpu":
			searcher, err = gpu.NewSearcher(maskImg)
		case "cpu":
			searcher, err = cpu.NewSearcher(*workerCount, cpu.Mask{ORad: 8, IRad: 1})
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		defer searcher.Destroy()

		// TODO: textual seeds
		seed, err := strconv.ParseInt(flag.Arg(0), 10, 64)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Could not convert seed to integer:", err)
			os.Exit(2)
		}

		searchRange64, err := strconv.ParseInt(flag.Arg(1), 10, 32)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Could not convert range to integer:", err)
			os.Exit(2)
		}
		searchRange := int32(searchRange64)
		if searchRange < 0 {
			fmt.Fprintln(os.Stderr, "Range must not be negative")
			os.Exit(2)
		}

		threshold64, err := strconv.ParseInt(flag.Arg(2), 10, 0)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Could not convert threshold to integer:", err)
			os.Exit(2)
		}
		threshold := int(threshold64)

		runSearch(searcher,
			int32(centerPos[0])-searchRange, int32(centerPos[1])-searchRange,
			int32(centerPos[0])+searchRange, int32(centerPos[1])+searchRange,
			threshold, seed,
		)

	case 2:
		// GUI mode
		// TODO: support CPU search
		seed, err := strconv.ParseInt(flag.Arg(0), 10, 64)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Could not convert seed to integer:", err)
			os.Exit(2)
		}

		threshold64, err := strconv.ParseInt(flag.Arg(1), 10, 0)
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

		app, err := NewApp(seed, threshold, centerPos, maskImg, *vsync)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		defer app.Destroy()
		app.Main()
	}
}
