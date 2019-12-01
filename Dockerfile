FROM golang:alpine as builder

ARG GOOS=linux
ARG GOARCH=amd64
ARG GOARM=""

WORKDIR /go/src
ADD . .
RUN CGO_ENABLED=0 GOOS=${GOOS} GOARCH=${GOARCH} GOARM=${GOARM} go build -mod vendor -tags netgo ./cmd/rc-arduino/




FROM gcr.io/distroless/static

USER 1234
COPY --from=builder /go/src/rc-arduino /go/bin/rc-arduino
ENTRYPOINT ["/go/bin/rc-arduino"]
