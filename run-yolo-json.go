package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// Run the YOLOv3 model over all test images.
// (if we labeled some then it could include some that we used for yolov3 training)

type Detection struct {
	X int
	Y int
	P int
}

func main() {
	darknetCfg := os.Args[1]
	darknetBackup := os.Args[2]
	shahinDir := os.Args[3]
	df := os.Args[4] // train, val, or test
	outPath := os.Args[5]

	c := exec.Command("./darknet", "detect", darknetCfg, darknetBackup, "-thresh", "0.02")
	c.Dir = "darknet/"
	stdin, err := c.StdinPipe()
	if err != nil {
		panic(err)
	}
	stderr, err := c.StderrPipe()
	if err != nil {
		panic(err)
	}
	stdout, err := c.StdoutPipe()
	if err != nil {
		panic(err)
	}
	go func() {
		r := bufio.NewReader(stderr)
		for {
			line, err := r.ReadString('\n')
			if err == io.EOF {
				break
			} else if err != nil {
				panic(err)
			}
			line = strings.TrimSpace(line)
			fmt.Println("[yolo] [stderr] " + line)
		}
	}()
	r := bufio.NewReader(stdout)
	if err := c.Start(); err != nil {
		panic(err)
	}

	getLines := func() []string {
		var output string
		for {
			line, err := r.ReadString(':')
			if err != nil {
				panic(err)
			}
			fmt.Println("[yolo] [stdout] " + strings.TrimSpace(line))
			output += line
			if strings.Contains(line, "Enter") {
				break
			}
		}
		return strings.Split(output, "\n")
	}
	parseLines := func(lines []string) []Detection {
		var detections []Detection
		for i := 0; i < len(lines); i++ {
			if !strings.Contains(lines[i], "%") {
				continue
			}
			confidence, _ := strconv.Atoi(strings.Trim(strings.Split(lines[i], ": ")[1], "%"))
			for !strings.Contains(lines[i], "Bounding Box:") {
				i++
			}
			parts := strings.Split(strings.Split(lines[i], ": ")[1], ", ")
			if len(parts) != 4 {
				panic(fmt.Errorf("bad bbox line %s", lines[i]))
			}
			var left, top, right, bottom int
			for _, part := range parts {
				kvsplit := strings.Split(part, "=")
				k := kvsplit[0]
				v, _ := strconv.Atoi(kvsplit[1])
				if k == "Left" {
					left = v
				} else if k == "Top" {
					top = v
				} else if k == "Right" {
					right = v
				} else if k == "Bottom" {
					bottom = v
				}
			}
			detections = append(detections, Detection{
				X: (left+right)/2,
				Y: (top+bottom)/2,
				P: confidence,
			})
		}
		return detections
	}
	getLines()

	bytes, err := ioutil.ReadFile(shahinDir + df + "_df.csv")
	if err != nil {
		panic(err)
	}
	fnames := strings.Split(string(bytes), "\n")
	detections := make(map[string][]Detection)
	for _, fname := range fnames {
		fname = strings.TrimSpace(fname)
		if !strings.HasSuffix(fname, ".jpg") {
			continue
		}
		fmt.Printf("[yolo] processing %s\n", fname)
		stdin.Write([]byte(shahinDir + "training_v1/" + fname + "\n"))
		lines := getLines()
		detections[fname] = parseLines(lines)
	}

	bytes, err = json.Marshal(detections)
	if err != nil {
		panic(err)
	}
	if err := ioutil.WriteFile(outPath, bytes, 0644); err != nil {
		panic(err)
	}
}
