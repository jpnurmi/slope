.PHONY: all build test vet clean

all: vet test build

build:
	go build .

test:
	go test -count=1 ./...

vet:
	go vet ./...

clean:
	rm -f slope slope.exe
