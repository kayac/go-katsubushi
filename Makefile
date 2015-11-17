GIT_VER := $(shell git describe --tags)

all: katsubushi

katsubushi: cmd/katsubushi/katsubushi

cmd/katsubushi/katsubushi: *.go cmd/katsubushi/*.go
	cd cmd/katsubushi && go build

.PHONEY: clean test packages
clean:
	rm -rf cmd/katsubushi/katsubushi pkg/*

test:
	go test

packages:
	cd cmd/katsubushi && gox -os="linux darwin" -arch="386 amd64" -output "../../pkg/${GIT_VER}-{{.OS}}-{{.Arch}}/{{.Dir}}"
	cd pkg && find * -type dir -exec ../pack.sh {} katsubushi \;
