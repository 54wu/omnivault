.PHONY: build test install clean

build:
	go build -o bin/omnivault ./cmd/omnivault

test:
	go test -v -race ./...

install: build
	install -m 755 bin/omnivault /usr/local/bin/omnivault

clean:
	rm -rf bin/
