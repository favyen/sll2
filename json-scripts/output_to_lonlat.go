package main

import (
	"github.com/mitroadmaps/gomapinfer/common"
	"github.com/mitroadmaps/gomapinfer/googlemaps"

	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strconv"
)

type Detection struct {
	X int
	Y int
	P int
}

func main() {
	shahinCSV := os.Args[1]
	predFname := os.Args[2]
	outFname := os.Args[3]

	imToTile := make(map[string][2]int)
	file, err := os.Open(shahinCSV)
	if err != nil {
		panic(err)
	}
	rd := csv.NewReader(file)
	for {
		records, err := rd.Read()
		if records != nil {
			fname := records[4]
			xtile, _ := strconv.Atoi(records[0])
			ytile, _ := strconv.Atoi(records[1])
			imToTile[fname] = [2]int{xtile, ytile}
		}
		if err == io.EOF {
			break
		} else if err != nil {
			panic(err)
		}
	}
	file.Close()

	bytes, err := ioutil.ReadFile(predFname)
	if err != nil {
		panic(err)
	}
	detections := make(map[string][]Detection)
	if err := json.Unmarshal(bytes, &detections); err != nil {
		panic(err)
	}

	outputs := make(map[string][][2]float64)
	for fname := range detections {
		tile, ok := imToTile[fname]
		if !ok {
			panic(fmt.Errorf("missing file %s", fname))
		}
		for _, detection := range detections[fname] {
			p := common.Point{float64(detection.X/2), float64(detection.Y/2)}
			p = googlemaps.MapboxToLonLat(p, 20, tile)
			outputs[fname] = append(outputs[fname], [2]float64{p.X, p.Y})
		}
	}

	bytes, err = json.Marshal(outputs)
	if err != nil {
		panic(err)
	}
	if err := ioutil.WriteFile(outFname, bytes, 0644); err != nil {
		panic(err)
	}
}
