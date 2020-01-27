.PHONY: test docker

DOCKER_IMG = cyrilix/robocar-camera

test:
	go test
	-mod vendor ./cmd/rc-camera ./camera

docker:
	docker buildx build . --platform linux/arm/7,linux/arm64,linux/amd64 -t ${DOCKER_IMG} --push

