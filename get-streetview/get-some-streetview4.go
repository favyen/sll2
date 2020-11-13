package main

import (
	"github.com/mitroadmaps/gomapinfer/common"
	coords "github.com/mitroadmaps/gomapinfer/googlemaps"

	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	//"math/rand"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
)

const MetaURL = "https://maps.googleapis.com/maps/api/streetview/metadata?size=512x512&location=%f,%f&fov=110&pitch=20&source=outdoor&key=%s"
const ImageURL = "https://maps.googleapis.com/maps/api/streetview?size=512x512&pano=%s&fov=%d&pitch=%d&heading=%d&key=%s"

type PanoMetadata struct {
	ID string `json:"pano_id"`
	Location struct {
		Latitude float64 `json:"lat"`
		Longitude float64 `json:"lng"`
	} `json:"location"`
	Status string `json:"status"`
	FOV int
	Pitch int
	Heading int
	Distance int
}

func GetMetadata(cameraPt common.Point, lightPt common.Point, key string) PanoMetadata {
	// get closest pano to cameraPt
	var metadata PanoMetadata
	url := fmt.Sprintf(MetaURL, cameraPt.Y, cameraPt.X, key)
	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}
	bytes, err := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		panic(fmt.Errorf("got status code %d: %s", resp.StatusCode, string(bytes)))
	}
	if err := json.Unmarshal(bytes, &metadata); err != nil {
		panic(err)
	}

	if metadata.Status != "OK" {
		return metadata
	}

	// determine heading
	// https://www.igismap.com/formula-to-find-bearing-or-heading-angle-between-two-points-latitude-longitude/
	panoRadians := common.Point{metadata.Location.Longitude, metadata.Location.Latitude}.Scale(math.Pi/180)
	lightRadians := lightPt.Scale(math.Pi/180)
	dlon := lightRadians.X - panoRadians.X
	dlat := lightRadians.Y - panoRadians.Y

	x := math.Cos(lightRadians.Y) * math.Sin(dlon)
	y := math.Cos(panoRadians.Y) * math.Sin(lightRadians.Y) - math.Sin(panoRadians.Y) * math.Cos(lightRadians.Y) * math.Cos(dlon)
	heading := math.Atan2(x, y)
	metadata.Heading = int(heading * 180/math.Pi)
	metadata.Heading = (metadata.Heading + 360) % 360

	haversineA := math.Sin(dlat/2) * math.Sin(dlat/2) + math.Cos(panoRadians.Y) * math.Cos(lightRadians.Y) * math.Sin(dlon/2) * math.Sin(dlon/2)
	haversineC := 2 * math.Atan2(math.Sqrt(haversineA), math.Sqrt(1 - haversineA))
	distance := 6371 * 1000 * haversineC
	metadata.Distance = int(distance)

	metadata.FOV = 110
	metadata.Pitch = 20
	return metadata
}

// Returns a 512x512 image centered at the point using the specified zoom level and API key
func GetStreetView(metadata PanoMetadata, key string, fname string) {
	url := fmt.Sprintf(ImageURL, metadata.ID, metadata.FOV, metadata.Pitch, metadata.Heading, key)
	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}
	if resp.StatusCode != 200 || (resp.Header.Get("Content-Type") != "image/png" && resp.Header.Get("Content-Type") != "image/jpeg") {
		var errdesc string
		if resp.Header.Get("Content-Type") != "image/png" && resp.Header.Get("Content-Type") != "image/jpeg" {
			if bytes, err := ioutil.ReadAll(resp.Body); err == nil {
				errdesc = string(bytes)
			}
		}
		if resp.StatusCode == 500 {
			fmt.Printf("warning: got 500 (errdesc=%s) on %s (retrying later)\n", errdesc, url)
			time.Sleep(time.Minute)
			GetStreetView(metadata, key, fname)
			return
		} else {
			panic(fmt.Errorf("got status code %d content type %s (errdesc=%s url=%s)", resp.StatusCode, resp.Header.Get("Content-Type"), errdesc, url))
		}
	}
	imBytes, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		panic(err)
	}
	if resp.Header.Get("Content-Type") == "image/png" {
		fname += ".png"
	} else {
		fname += ".jpg"
	}
	if err := ioutil.WriteFile(fname, imBytes, 0644); err != nil {
		panic(err)
	}

	metaBytes, err := json.Marshal(metadata)
	if err != nil {
		panic(err)
	}
	if err := ioutil.WriteFile(fname + ".json", metaBytes, 0644); err != nil {
		panic(err)
	}
}

func main() {
	predFname := os.Args[1]
	graphFname := os.Args[2]
	key := os.Args[3]
	outPath := os.Args[4]

	// load predicted points
	bytes, err := ioutil.ReadFile(predFname)
	if err != nil {
		panic(err)
	}
	pred := make(map[string][][2]float64)
	if err := json.Unmarshal(bytes, &pred); err != nil {
		panic(err)
	}
	var fnames []string
	for fname := range pred {
		fnames = append(fnames, fname)
	}
	sort.Strings(fnames)

	// load road network graph
	log.Println("loading graph")
	g, err := common.ReadGraph(graphFname)
	if err != nil {
		panic(err)
	}
	for _, node := range g.Nodes {
		node.Point = coords.LonLatToMeters(node.Point)
	}
	log.Println("loading index")
	gIdx := g.GridIndex(128)
	log.Println("done loading")

	// find closest point on road network to light
	// but point must be at least 10 from any blacklisted point (previously used)
	findClosestPoint := func(p common.Point, blacklist []common.Point) *common.Point {
		isBlacklisted := func(try common.Point) bool {
			for _, b := range blacklist {
				if try.Distance(b) < 10 {
					return true
				}
			}
			return false
		}

		var bestPoint common.Point
		var bestDistance float64 = -1
		for _, edge := range gIdx.Search(p.Bounds().AddTol(64)) {
			edgePos := edge.ClosestPos(p)
			// try different points on this edge to get around the blacklist
			for _, offset := range []float64{0, -11, 11, -22, 22} {
				npos := common.EdgePos{edgePos.Edge, edgePos.Position+offset}
				if npos.Position < 0 {
					npos.Position = 0
				} else if npos.Position > npos.Edge.Segment().Length() {
					npos.Position = npos.Edge.Segment().Length()
				}
				try := npos.Point()
				if isBlacklisted(try) {
					continue
				}
				d := try.Distance(p)
				if bestDistance == -1 || d < bestDistance {
					bestPoint = try
					bestDistance = d
				}
				break
			}
		}
		if bestDistance == -1 {
			return nil
		}
		return &bestPoint
	}

	//for _, idx := range rand.Perm(len(fnames))[0:64] {
	//	fname := fnames[idx]
	for _, fname := range fnames {
		if len(pred[fname]) == 0 {
			continue
		}
		label := strings.Split(fname, ".")[0]
		for i, p := range pred[fname] {
			// find closest road
			lightPt := common.Point{p[0], p[1]}
			pMeters := coords.LonLatToMeters(lightPt)

			// fetch 5 images at different locations around pos point
			// all pairs of points should be at least 10 m apart (and correspond to a different pano ID)
			seenPanoIDs := make(map[string]bool)
			var blacklist []common.Point
			for j := 0; j < 5; j++ {
				closestMeters := findClosestPoint(pMeters, blacklist)
				if closestMeters == nil {
					break
				}
				blacklist = append(blacklist, *closestMeters)
				cameraPt := coords.MetersToLonLat(*closestMeters)

				metadata := GetMetadata(cameraPt, lightPt, key)
				if metadata.Status != "OK" {
					fmt.Printf("skip %s:%v due to status %s\n", label, p, metadata.Status)
					continue
				} else if seenPanoIDs[metadata.ID] {
					continue
				}
				seenPanoIDs[metadata.ID] = true
				fmt.Printf("get %s:%v\n", label, p)
				outFname := fmt.Sprintf("%s/%s-%d-%d", outPath, label, i, j)
				if _, err := os.Stat(outFname + ".jpg"); err == nil {
					fmt.Printf("skip %s:%v already exists\n", label, p)
					continue
				}
				GetStreetView(metadata, key, outFname)
			}
		}
	}
}
