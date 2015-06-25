all: katsubushi

katsubushi: cmd/katsubushi/katsubushi

cmd/katsubushi/katsubushi: *.go cmd/katsubushi/*.go
	cd cmd/katsubushi && go build

.PHONEY: clean test
clean:
	cmd/katsubushi/katsubushi

test:
	go test
