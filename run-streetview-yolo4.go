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

/*
# python script to incorporate the streetview detection outputs
import json
with open('outputs/yolo-shahin-out-thresh10.json', 'r') as f:
	detections = json.load(f)
with open('streetview-outputs/yolo-shahin-out-thresh10/out-bulb-and-pole4.json', 'r') as f:
	sv_detections = json.load(f)
good = set()
for label, dlist in sv_detections.items():
	if not dlist:
		continue
	dlist = [d for d in dlist if d['Class'] == 'bicycle']
	ok = True
	for d in dlist:
		height = d['Bottom'] - d['Top']
		offset = abs((d['Left']+d['Right'])//2-256)
		'''if height > 300:
			ok = True
		elif height > 200 and offset < 150:
			ok = True
		elif height > 100 and offset < 75:
			ok = True
		elif height > 50 and offset < 50:
			ok = True'''
		if offset < 50:
			ok = True
			break
		factor = float(height)/float(offset)
		if factor > 1:
			ok = True
			break
	if not ok:
		continue
	parts = label.split('-')
	fname = parts[0] + '.jpg'
	idx = int(parts[1])
	good.add((fname, idx))
ndetections = {}
eliminated = []
for fname, dlist in detections.items():
	if not dlist:
		ndetections[fname] = []
		continue
	nlist = []
	for idx, d in enumerate(dlist):
		if (fname, idx) not in good:
			print fname, idx
			eliminated.append((fname, idx))
			continue
		nlist.append(d)
	ndetections[fname] = nlist
with open('outputs/yolo-shahin-out-thresh10-svfilter-bulb-and-pole4.json', 'w') as f:
	json.dump(ndetections, f)
*/

/*
# python script to visualize the removed detections from script above
l = eliminated # ...
import json
import os
import skimage.io
with open('outputs/yolo-shahin-out-thresh10.json', 'r') as f:
	lights = json.load(f)
with open('streetview-outputs/yolo-shahin-out-thresh10/out-bulb-and-pole4.json', 'r') as f:
	sv_detections = json.load(f)
for fname, idx in l:
	label = fname.split('.jpg')[0]
	sv_fname = 'streetview-outputs/yolo-shahin-out-thresh10/images/{}-{}.jpg'.format(label, idx)
	if not os.path.exists(sv_fname):
		continue
	# create satellite image visualization
	im = skimage.io.imread('/mnt/signify/la/shahin-dataset/training_v1/{}'.format(fname))
	p = lights[fname][idx]
	x, y = p['X'], p['Y']
	if x < 33:
		x = 33
	if x > im.shape[1]-33:
		x = im.shape[1]-33
	if y < 33:
		y = 33
	if y > im.shape[0]-33:
		y = im.shape[0]-33
	im[y-32:y+32, x-33:x-31, :] = [255, 0, 0]
	im[y-32:y+32, x+31:x+33, :] = [255, 0, 0]
	im[y-33:y-31, x-32:x+32, :] = [255, 0, 0]
	im[y+31:y+33, x-32:x+32, :] = [255, 0, 0]
	skimage.io.imsave('/home/ubuntu/vis/{}_{}_sat.jpg'.format(label, idx), im)
	# create street view visualizations
	svcounter = 0
	for sv_fname in os.listdir('streetview-outputs/yolo-shahin-out-thresh10/images-get4/'):
		if not sv_fname.endswith('.jpg') or not sv_fname.startswith('{}-{}-'.format(label, idx)):
			continue
		im = skimage.io.imread('streetview-outputs/yolo-shahin-out-thresh10/images-get4/' + sv_fname)
		if sv_fname.split('.')[0] in sv_detections and sv_detections[sv_fname.split('.')[0]]:
			for d in sv_detections[sv_fname.split('.')[0]]:
				if d['Class'] != 'bicycle':
					continue
				im[d['Top']:d['Bottom'], d['Left']:d['Left']+2, :] = [255, 0, 0]
				im[d['Top']:d['Bottom'], d['Right']-2:d['Right'], :] = [255, 0, 0]
				im[d['Top']:d['Top']+2, d['Left']:d['Right'], :] = [255, 0, 0]
				im[d['Bottom']-2:d['Bottom'], d['Left']:d['Right'], :] = [255, 0, 0]
		skimage.io.imsave('/home/ubuntu/vis/{}_{}_{}_streetview.jpg'.format(label, idx, svcounter), im)
		svcounter += 1
*/

type Detection struct {
	Left int
	Top int
	Right int
	Bottom int
	P int
	Class string
}

func main() {
	darknetCfg := os.Args[1]
	darknetBackup := os.Args[2]
	inPath := os.Args[3]
	outPath := os.Args[4]

	c := exec.Command("./darknet", "detect", darknetCfg, darknetBackup, "-thresh", "0.1")
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
			parts := strings.Split(lines[i], ": ")
			var detection Detection
			detection.Class = parts[0]
			detection.P, _ = strconv.Atoi(strings.Trim(parts[1], "%"))
			for !strings.Contains(lines[i], "Bounding Box:") {
				i++
			}
			parts = strings.Split(strings.Split(lines[i], ": ")[1], ", ")
			if len(parts) != 4 {
				panic(fmt.Errorf("bad bbox line %s", lines[i]))
			}
			for _, part := range parts {
				kvsplit := strings.Split(part, "=")
				k := kvsplit[0]
				v, _ := strconv.Atoi(kvsplit[1])
				if k == "Left" {
					detection.Left = v
				} else if k == "Top" {
					detection.Top = v
				} else if k == "Right" {
					detection.Right = v
				} else if k == "Bottom" {
					detection.Bottom = v
				}
			}
			detections = append(detections, detection)
		}
		return detections
	}
	getLines()

	files, err := ioutil.ReadDir(inPath)
	if err != nil {
		panic(err)
	}
	detections := make(map[string][]Detection)
	for _, fi := range files {
		if !strings.HasSuffix(fi.Name(), ".jpg") {
			continue
		}
		fmt.Printf("[yolo] processing %s\n", fi.Name())
		stdin.Write([]byte(inPath + fi.Name() + "\n"))
		lines := getLines()
		label := strings.Split(fi.Name(), ".")[0]
		detections[label] = parseLines(lines)
	}

	bytes, err := json.Marshal(detections)
	if err != nil {
		panic(err)
	}
	if err := ioutil.WriteFile(outPath, bytes, 0644); err != nil {
		panic(err)
	}
}
