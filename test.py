import json
import math
import numpy
import sys

def distance(p1,p2):
	a = p1[0]-p2[0]
	b = p1[1]-p2[1]

	return math.sqrt(a*a + b*b)

def match(pred, gt, d_thr = 96):
	matched_gt = {}
	matched_pred = {}

	def good_func(p):
		return p[0] > 64 and p[0] < 448 and p[1] > 64 and p[1] < 448
	pred_good = [p for p in pred if good_func(p)]
	gt_good = [p for p in gt if good_func(p)]

	used = set()
	for p in pred_good:
		best = None
		best_distance = None
		for g in gt:
			if g in used:
				continue
			d = distance(p, g)
			if d > d_thr:
				continue
			if best is None or d < best_distance:
				best = g
				best_distance = d

		if best is not None:
			used.add(best)
			matched_pred[p] = True

	used = set()
	for g in gt_good:
		best = None
		best_distance = None
		for p in pred:
			if p in used:
				continue
			d = distance(g, p)
			if d > d_thr:
				continue
			if best is None or d < best_distance:
				best = p
				best_distance = d

		if best is not None:
			used.add(best)
			matched_gt[g] = True

	return pred_good, gt_good, matched_pred, matched_gt

if __name__ == "__main__":
	gt_json = sys.argv[1]
	pred_json = sys.argv[2]
	test_df_fname = sys.argv[3]

	with open(gt_json, 'r') as f:
		gt_dict = json.load(f)
	with open(pred_json, 'r') as f:
		pred_dict = json.load(f)
	with open(test_df_fname) as f:
		lines = f.readlines()
		lines = [line.strip() for line in lines if line.strip()]
		fnames = [line for line in lines if '.jpg' in line]

	for thr in [2, 3, 4, 5, 7, 10, 20, 30, 40, 50, 60, 70, 80, 90]:
	#for thr in [1000, 2000, 3000, 4000, 5000, 6000, 7000, 8000, 9000, 9500, 9600, 9700, 9800, 9900, 9950, 9975, 9990]:
		tot_gt = 0
		tot_gt_matched = 0
		tot_prop = 0
		tot_prop_matched = 0

		for fname in fnames:
			if fname not in gt_dict:
				continue
			gt = gt_dict[fname]
			if len(gt) == 0:
				continue
			gt = [tuple(g) for g in gt]
			pred = []
			if pred_dict.get(fname, None):
				for p in pred_dict[fname]:
					pred.append((p['X'], p['Y'], p['P']))

			pred = [(p[0], p[1]) for p in pred if p[2] >= thr]
			pred_good, gt_good, matched_pred, matched_gt = match(pred, gt)
			tot_gt += len(gt_good)
			tot_gt_matched += len(matched_gt)
			tot_prop += len(pred_good)
			tot_prop_matched += len(matched_pred)

		p = tot_prop_matched/tot_prop
		r = tot_gt_matched/tot_gt
		f1 = 2*p*r/(p+r+0.00001)
		print('threshold %d p=%.3f r=%.3f f1=%.3f (%d, %d)' % (thr, p, r, f1, tot_prop, tot_gt))

'''
YOLOv3:
threshold 2 p=0.546 r=0.905 f1=0.681
threshold 3 p=0.591 r=0.898 f1=0.713
threshold 4 p=0.651 r=0.878 f1=0.747
threshold 5 p=0.695 r=0.855 f1=0.767
threshold 7 p=0.755 r=0.831 f1=0.791
threshold 10 p=0.806 r=0.792 f1=0.799
threshold 20 p=0.868 r=0.657 f1=0.748
threshold 30 p=0.913 r=0.562 f1=0.696
threshold 40 p=0.929 r=0.468 f1=0.622
threshold 50 p=0.952 r=0.381 f1=0.544
threshold 60 p=0.949 r=0.293 f1=0.448
threshold 70 p=0.983 r=0.212 f1=0.348
threshold 80 p=0.985 r=0.119 f1=0.212
threshold 90 p=1.000 r=0.042 f1=0.081

Mask R-CNN:
threshold 1000 p=0.268 r=0.961 f1=0.419
threshold 2000 p=0.268 r=0.961 f1=0.419
threshold 3000 p=0.268 r=0.961 f1=0.419
threshold 4000 p=0.268 r=0.961 f1=0.419
threshold 5000 p=0.268 r=0.961 f1=0.419
threshold 6000 p=0.283 r=0.954 f1=0.437
threshold 7000 p=0.299 r=0.949 f1=0.455
threshold 8000 p=0.321 r=0.944 f1=0.479
threshold 9000 p=0.356 r=0.931 f1=0.515
threshold 9500 p=0.393 r=0.923 f1=0.551
threshold 9600 p=0.405 r=0.917 f1=0.562
threshold 9700 p=0.421 r=0.903 f1=0.575
threshold 9800 p=0.447 r=0.888 f1=0.595
threshold 9900 p=0.508 r=0.859 f1=0.638
threshold 9950 p=0.577 r=0.813 f1=0.675
threshold 9975 p=0.643 r=0.758 f1=0.696
threshold 9990 p=0.751 r=0.587 f1=0.659
'''
