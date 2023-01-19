build:
	go build -buildvcs=false -o ./bin/goblockchain 

all:
	go build ./... 

run: build
	./bin/goblockchain

test:
	go test  ./... -race
