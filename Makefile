.PHONY: test docker

DOCKER_IMG = cyrilix/robocar-arduino

test:
	go test -mod vendor ./cmd/rc-arduino ./arduino

docker:
	export DOCKEC_CLI_EXPERIMENTAL=enabled
	docker buildx build . --platform linux/arm/7,linux/arm64,linux/amd64 -t ${DOCKER_IMG} --push

