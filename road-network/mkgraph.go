package main

import (
	"github.com/mitroadmaps/gomapinfer/common"
)

func main() {
	boundList := []common.Rectangle{{
		common.Point{-118.50, 33.96},
		common.Point{-118.19, 34.11},
	}}
	graphs, err := common.LoadOSMMultiple("/data/discover-datasets/osm/jul2020-california.osm.pbf", boundList, common.OSMOptions{
		Verbose: true,
	})
	if err != nil {
		panic(err)
	}
	graph := graphs[0]
	if err := graph.Write("la.graph"); err != nil {
		panic(err)
	}
}
