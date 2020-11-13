# python script to incorporate the streetview detection outputs
import csv
import json
import math
import numpy
import os
import skimage.io
from discoverlib import coords, geom

SV_WIDTH = 512.0

with open('outputs/yolo-new-thresh3.json', 'r') as f:
	detections = json.load(f)
with open('streetview-outputs/yolo-new-thresh3/out-bulb-and-pole4.json', 'r') as f:
	sv_detections = json.load(f)

# get satellite image metadata mapping from fname to web mercator tile
# copied from output_to_lonlat.go
im_to_tile = {}
with open('/mnt/signify/la/shahin-dataset/training_v1.csv', 'r') as f:
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
	with open('streetview-outputs/yolo-new-thresh3/images-get4/{}.jpg.json'.format(label)) as f:
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

		# visualization code
		'''sat_im = skimage.io.imread('/mnt/signify/la/shahin-dataset/training_v1/{}'.format(fname))
		sat_d = detections[fname][sat_idx]
		draw_point(sat_im, sat_d['X'], sat_d['Y'], 32, [255, 0, 0])
		big_im = numpy.zeros((1024, 1024, 3), dtype='uint8')
		big_im[256:768, 256:768, :] = sat_im
		draw_point(big_im, estimate.x+256, estimate.y+256, 32, [255, 255, 0])
		draw_point(big_im, camera_position.x+256, camera_position.y+256, 32, [0, 255, 0])
		skimage.io.imsave('/home/ubuntu/vis/{}_sat.jpg'.format(label), big_im)
		sv_im = skimage.io.imread('streetview-outputs/yolo-shahin-out-thresh10/images-get4/{}.jpg'.format(label))
		draw_rect(sv_im, sv_d['Left'], sv_d['Top'], sv_d['Right'], sv_d['Bottom'], [255, 0, 0])
		skimage.io.imsave('/home/ubuntu/vis/{}_sv.jpg'.format(label), sv_im)'''

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
			print fname, idx
			eliminated.append((fname, idx))
			continue
		nlist.append(d)
	ndetections[fname] = nlist
with open('outputs/yolo-new-thresh3-svfilter-bulb-and-pole4.json', 'w') as f:
	json.dump(ndetections, f)

# optional: visualize the removed detections from script above
for fname, idx in eliminated:
	label = fname.split('.jpg')[0]
	# create satellite image visualization
	im = skimage.io.imread('/mnt/signify/la/shahin-dataset/training_v1/{}'.format(fname))
	for j, p in enumerate(detections[fname]):
		if j == idx:
			color = [255, 0, 0]
		else:
			color = [255, 255, 0]
		draw_point(im, p['X'], p['Y'], 32, color)
	skimage.io.imsave('/home/ubuntu/vis/{}_{}_sat.jpg'.format(label, idx), im)
	# create street view visualizations
	svcounter = 0
	for sv_fname in os.listdir('streetview-outputs/yolo-new-thresh3/images-get4/'):
		if not sv_fname.endswith('.jpg') or not sv_fname.startswith('{}-{}-'.format(label, idx)):
			continue
		im = skimage.io.imread('streetview-outputs/yolo-new-thresh3/images-get4/' + sv_fname)
		if sv_fname.split('.')[0] in sv_detections and sv_detections[sv_fname.split('.')[0]]:
			for d in sv_detections[sv_fname.split('.')[0]]:
				if d['Class'] != 'bicycle':
					continue
				draw_rect(im, d['Left'], d['Top'], d['Right'], d['Bottom'], [255, 0, 0])
		skimage.io.imsave('/home/ubuntu/vis/{}_{}_{}_streetview.jpg'.format(label, idx, svcounter), im)
		svcounter += 1
