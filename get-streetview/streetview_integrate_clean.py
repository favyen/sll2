# python script to incorporate the streetview detection outputs
import csv
import json
import math
import numpy
import os
import skimage.io
import sys
from discoverlib import coords, geom

SV_WIDTH = 512.0

sat_det_path = sys.argv[1]
sv_det_path = sys.argv[2]
sv_image_path = sys.argv[3]
csv_path = sys.argv[4] # training_v1.csv
out_path = sys.argv[5]

with open(sat_det_path, 'r') as f:
	detections = json.load(f)
with open(sv_det_path, 'r') as f:
	sv_detections = json.load(f)

# get satellite image metadata mapping from fname to web mercator tile
# copied from output_to_lonlat.go
im_to_tile = {}
with open(csv_path, 'r') as f:
	rd = csv.reader(f)
	for row in rd:
		if row[0] == 'xtile':
			continue
		fname = row[4]
		xtile = int(row[0])
		ytile = int(row[1])
		im_to_tile[fname] = (xtile, ytile)

def clip(x, lo, hi):
	if x < lo:
		return lo
	if x > hi:
		return hi
	return x
def draw_point(im, x, y, side, color):
	draw_rect(im, x-side, y-side, x+side, y+side, color)
def draw_rect(im, sx, sy, ex, ey, color):
	sx = clip(int(sx), 2, im.shape[1]-2)
	sy = clip(int(sy), 2, im.shape[0]-2)
	ex = clip(int(ex), 2, im.shape[1]-2)
	ey = clip(int(ey), 2, im.shape[0]-2)
	im[sy:ey, sx:sx+2, :] = color
	im[sy:ey, ex-2:ex, :] = color
	im[sy:sy+2, sx:ex, :] = color
	im[ey-2:ey, sx:ex, :] = color

# go through the streetview detections and match each detection to the closest
# satellite image detection. we mark the sat detection good only if the sv
# detection was supposed to be for that sat detection, and it is closest.
good = set()
for label, dlist in sv_detections.items():
	if not dlist:
		continue

	parts = label.split('-')
	fname = parts[0] + '.jpg'
	sat_idx = int(parts[1])
	sv_idx = int(parts[2])
	mapbox_tile = im_to_tile[fname]

	# get the metadata for this streetview image
	# this tells us the camera location and heading
	with open('{}/{}.jpg.json'.format(sv_image_path, label)) as f:
		sv_meta = json.load(f)
	camera_position = geom.FPoint(sv_meta['location']['lng'], sv_meta['location']['lat'])
	camera_position = coords.lonLatToMapbox(camera_position, 20, mapbox_tile)
	camera_position = camera_position.scale(2) # since we use 512x512 instead of standard 256x256 tiles
	camera_heading = (90-sv_meta['Heading'])*math.pi/180
	angle_per_pixel = (sv_meta['FOV']*math.pi/180)/SV_WIDTH

	dlist = [d for d in dlist if d['Class'] == 'bicycle']
	ok = False
	for sv_d in dlist:
		# compute distance from height
		height = sv_d['Bottom'] - sv_d['Top']
		distance = int(-0.6*height+295)

		# compute estimated location of this detection
		# need to account for inverted image y axis when computing unit vector
		# also the x axis is kind of inverted: higher x means further clockwise not counterclockwise
		cx = (sv_d['Left']+sv_d['Right'])/2
		cur_angle = camera_heading + angle_per_pixel*(SV_WIDTH/2-cx)
		unit_vector = geom.FPoint(math.cos(cur_angle), -math.sin(cur_angle))
		vector = unit_vector.scale(distance)
		estimate = camera_position.add(vector)

		# find closest sat detection
		closest_idx = None
		closest_distance = None
		for j, sat_d in enumerate(detections[fname]):
			dx = sat_d['X']-estimate.x
			dy = sat_d['Y']-estimate.y
			d = math.sqrt(dx*dx+dy*dy)
			if closest_idx is None or d < closest_distance:
				closest_idx = j
				closest_distance = d
		if closest_idx != sat_idx or closest_distance > 150:
			continue

		ok = True
		break

	if ok:
		good.add((fname, sat_idx))

# go through the sat detections and eliminate the ones that didn't get matched
ndetections = {}
eliminated = []
for fname, dlist in detections.items():
	if not dlist:
		ndetections[fname] = []
		continue
	nlist = []
	for idx, d in enumerate(dlist):
		if (fname, idx) not in good and d['P'] < 50:
			print(fname, idx)
			eliminated.append((fname, idx))
			continue
		nlist.append(d)
	ndetections[fname] = nlist
with open(out_path, 'w') as f:
	json.dump(ndetections, f)
