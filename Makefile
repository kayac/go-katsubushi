all: idg

idg: cmd/idg/idg

cmd/idg/idg: *.go cmd/idg/*.go
	cd cmd/idg && go build

.PHONEY: clean test
clean:
	cmd/idg/idg

test:
	go test
