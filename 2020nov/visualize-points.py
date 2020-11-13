import csv
import io
import json
import random
import skimage.io, skimage.draw
import sys
import urllib.request

api_key = sys.argv[1]

rows = []

with open('fixture_style_mit.csv', 'r') as f:
    rd = csv.DictReader(f)
    for row in rd:
        rows.append(row)

rows = random.sample(rows, 10)

for i, row in enumerate(rows):
    url = 'https://maps.googleapis.com/maps/api/streetview?size=600x400&location={},{}&key={}&heading={}&source=outdoor'.format(row['Latitude'], row['Longitude'], api_key, row['Direction'])
    with urllib.request.urlopen(url) as resp:
        s = resp.read()
    im = skimage.io.imread(io.BytesIO(s))

    x_list = json.loads(row['PointsX'])
    y_list = json.loads(row['PointsY'])
    #for j in range(len(x_list)):
    #    x, y = x_list[j], y_list[j]
    #    im[y-1:y+1, x-1:x+1, :] = [255, 0, 0]
    rr, cc = skimage.draw.polygon(y_list, x_list)
    im[rr, cc, :] = [255, 0, 0]
    print(i, x_list, y_list)
    skimage.io.imsave('/tmp/x/{}.jpg'.format(i), im)

# do unsupervised clustering on satellite image detections (e.g. over the yolov3 hidden representation)
# code for getting the streetview images
# code/model the yolov3 for satellite image and for streetview
# implement shahin's idea for unsupervised clustering: argmin(G_w(f_{k,m}) - G_w(f_{i,j}))_2
