.PHONY: test build build-keenetic package-keenetic

VERSION ?= dev
LDFLAGS = -s -w -X main.version=$(VERSION)

test:
	go test ./...

build:
	mkdir -p dist
	go build -trimpath -ldflags "$(LDFLAGS)" -o dist/keenups ./cmd/keenups

build-keenetic:
	mkdir -p dist
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags "$(LDFLAGS)" -o dist/keenups-linux-arm64 ./cmd/keenups

package-keenetic: build-keenetic
	./scripts/build-ipk.sh "$(VERSION)" dist/keenups-linux-arm64 dist
