.PHONY: test build build-keenetic

test:
	go test ./...

build:
	mkdir -p dist
	go build -trimpath -ldflags "-s -w" -o dist/keenups ./cmd/keenups

build-keenetic:
	mkdir -p dist
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags "-s -w" -o dist/keenups-linux-arm64 ./cmd/keenups
