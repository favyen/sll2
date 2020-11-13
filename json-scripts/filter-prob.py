import json
import sys

in_fname = sys.argv[1]
threshold = int(sys.argv[2])
out_fname = sys.argv[3]

with open(in_fname, 'r') as f:
	detections = json.load(f)

ndetections = {}
for fname, dlist in detections.items():
    if not dlist:
        ndetections[fname] = []
        continue
    nlist = []
    for d in dlist:
        if d['P'] < threshold:
            continue
        nlist.append(d)
    ndetections[fname] = nlist

with open(out_fname, 'w') as f:
    json.dump(ndetections, f)
