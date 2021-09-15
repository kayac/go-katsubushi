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
		goxz -pv="v${GIT_VER}" \
			-build-ldflags="-s -w -X github.com/kayac/go-katsubushi.Version=${GIT_VER}" \
			-os=darwin,linux \
			-arch=amd64,arm64 \
			-d=dist \
			./cmd/katsubushi

release:
	ghr -u kayac -r go-katsubushi -n "v${GIT_VER}" ${GIT_VER} dist/

docker: clean packages
	cd dist && \
		tar xvf go-katsubushi_v${GIT_VER}_linux_amd64.tar.gz && \
		tar xvf go-katsubushi_v${GIT_VER}_linux_arm64.tar.gz
	docker buildx build \
		--build-arg VERSION=v${GIT_VER} \
		--platform linux/amd64,linux/arm64 \
		-f docker/Dockerfile \
		-t katsubushi/katsubushi:v${GIT_VER} \
		.

docker-push: docker
    docker push katsubushi/katsubushi:v${GIT_VER}
