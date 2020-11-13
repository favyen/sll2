Suppose data is organized like this:

- Satellite image YOLOv3 cfg and model are in data/yolov3/
- Streetview YOLOv3 cfg and model are in data/yolov3sv/
- Satellite image dataset CSVs are e.g. data/dataset/train_df.csv and images are data/dataset/training_v1/...
- Labels annotated from satellite image + streetview are at data/manual41.json
- la.graph is in data/la.graph (generated by road-network/mkgraph.go)

First, apply YOLOv3 model on satellite images (test set):

	go run run-yolo-json.go data/yolov3/yolov3.cfg data/yolov3/yolov3.best data/dataset/ test sat-output.json
	python3 test.py data/manual41.json sat-output.json data/dataset/test_df.csv

Next, threshold the detections and convert them from pixel coordinates to longitude-latitude.
This is needed to fetch images from streetview API.

	python3 json-scripts/filter-prob.py sat-output.json 10 sat-output-thresholded.json
	go run json-scripts/output_to_lonlat.go data/dataset/training_v1.csv sat-output-thresholded.json sat-output-thresholded-lonlat.json

Now we can fetch a few streetview images for each street light detection:

	mkdir sv-images/
	go run get-some-streetview4.go sat-output-thresholded-lonlat.json [Google API Key] sv-images/

And then run YOLOv3 on streetview:

	go run run-streetview-yolo4.go data/yolov3sv/yolov3.cfg data/yolov3sv/yolov3.best sv-images/ sv-detections.json

Then there is some code get-streetview/streetview_integrate4b.py that can be adapted to combine satellite image detections with streetview detections.
