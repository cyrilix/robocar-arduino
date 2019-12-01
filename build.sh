#! /bin/bash

export IMG_NAME=cyrilix/robocar-arduino
export DOCKER_CLI_EXPERIMENTAL=enabled

set -e



docker build --no-cache --build-arg=GOOS=linux --build-arg=GOARCH=amd64 --build-arg=GOARM="" -f Dockerfile -t ${IMG_NAME}:amd64 .
docker build --no-cache --build-arg=GOOS=linux --build-arg=GOARCH=amd64 --build-arg=GOARM="" -f Dockerfile -t ${IMG_NAME}:arm64v8 .
docker build --no-cache --build-arg=GOOS=linux --build-arg=GOARCH=arm --build-arg=GOARM="7" -f Dockerfile -t ${IMG_NAME}:armv7 .
docker build --no-cache --build-arg=GOOS=linux --build-arg=GOARCH=arm --build-arg=GOARM="6" -f Dockerfile -t ${IMG_NAME}:armv6 .

if [[ -n "${DOCKER_PASSWORD}" && -n ${DOCKER_USERNAME} ]]
then
  echo "$DOCKER_PASSWORD" | docker login -u "$DOCKER_USERNAME" --password-stdin
  docker push "${IMG_NAME}:amd64"
  docker push "${IMG_NAME}:arm64v8"
  docker push "${IMG_NAME}:armv7"
  docker push "${IMG_NAME}:armv6"
fi

docker -D manifest create "${IMG_NAME}:latest" "${IMG_NAME}:amd64" "${IMG_NAME}:arm64v8" "${IMG_NAME}:armv7" "${IMG_NAME}:armv6"
docker -D manifest annotate "${IMG_NAME}:latest" "${IMG_NAME}:armv6" --os=linux --arch=arm --variant=v6
docker -D manifest annotate "${IMG_NAME}:latest" "${IMG_NAME}:armv7" --os=linux --arch=arm --variant=v7
docker -D manifest annotate "${IMG_NAME}:latest" "${IMG_NAME}:arm64v8" --os=linux --arch=arm64 --variant=v8


if [[ -n "${DOCKER_PASSWORD}" && -n ${DOCKER_USERNAME} ]]
then
  docker -D manifest push --purge "${IMG_NAME}:latest"
fi

