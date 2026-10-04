.PHONY: build build-zyxel run test vet clean

BINARY=mdc

build:
	go build -o $(BINARY) .

build-zyxel:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=5 go build -o $(BINARY) .

run: build
	./$(BINARY)

test:
	go test ./...

vet:
	go vet ./...

clean:
	rm -f $(BINARY)
