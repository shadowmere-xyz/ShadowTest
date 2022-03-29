TAG?=$(shell git describe --tags --always --dirty)
IMAGE_NAME?=guamulo/shadowtest:$(TAG)

.PHONY:
start_test_server:
	- pkill ss-server
	ss-server -v -p 6276 -k password &

.PHONY:
test: start_test_server
	go test ./... -count=1

.PHONY:
build_image:
	docker build -t $(IMAGE_NAME) .
