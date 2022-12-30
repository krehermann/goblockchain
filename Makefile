build:
	go build -buildvcs=false -o ./bin/goblockchain 

run: build
	./bin/goblockchain

test:
	go test -v ./...
