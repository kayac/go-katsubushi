GIT_VER := $(shell git describe --tags | sed -e 's/^v//')
export GO111MODULE := on

all: katsubushi

katsubushi: cmd/katsubushi/katsubushi

cmd/katsubushi/katsubushi: *.go cmd/katsubushi/*.go
	cd cmd/katsubushi && go build -ldflags "-w -s -X github.com/kayac/go-katsubushi.Version=${GIT_VER}"


.PHONEY: clean test packages install
install: cmd/katsubushi/katsubushi
	install cmd/katsubushi/katsubushi ${GOPATH}/bin

clean:
	rm -rf cmd/katsubushi/katsubushi dist/*

test:
	go test -race

packages:
	CGO_ENABLED=0 \
		goxz -pv="${GIT_VER}" \
			-build-ldflags="-s -w -X github.com/kayac/go-katsubushi.Version=${GIT_VER}" \
			-os=darwin,linux \
			-arch=amd64 \
			-d=dist \
			./cmd/katsubushi

release:
	ghr -u kayac -r go-katsubushi -n "$(GIT_VER)" $(GIT_VER) dist/
