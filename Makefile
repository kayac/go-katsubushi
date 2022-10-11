GIT_VER := $(shell git describe --tags | sed -e 's/^v//')
export GO111MODULE := on

all: grpc-gen katsubushi

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
	goreleaser build --skip-validate --rm-dist

packages-snapshot:
	goreleaser build --skip-validate --rm-dist --snapshot

docker: clean packages
	mv dist/go-katsubushi_linux_amd64_v1 dist/go-katsubushi_linux_amd64
	docker buildx build \
		--build-arg VERSION=v${GIT_VER} \
		--platform linux/amd64,linux/arm64 \
		-f docker/Dockerfile \
		-t katsubushi/katsubushi:v${GIT_VER} \
		-t ghcr.io/kayac/go-katsubushi:v${GIT_VER} \
		.

docker-push: docker
	docker buildx build \
		--build-arg VERSION=v${GIT_VER} \
		--platform linux/amd64,linux/arm64 \
		-f docker/Dockerfile \
		-t katsubushi/katsubushi:v${GIT_VER} \
		-t ghcr.io/kayac/go-katsubushi:v${GIT_VER} \
		--push \
		.

grpc-gen: proto/*.proto
	protoc -I=proto --go_out=./grpc --go-grpc_out=./grpc proto/*.proto
	mv grpc/katsubushi/grpc/*.go grpc
	rm -fr grpc/katsubushi
	protoc -I=proto --doc_out=./grpc --doc_opt=markdown,README.md proto/*.proto
