BINARY = eye

SRCS = %.go

all: $(BINARY)

$(BINARY): deps **/**/*.go **/*.go *.go
	go build -ldflags "-X main.buildCommit=`git describe --long --tags --dirty --always`" .

deps:
	go get ./...

build: $(BINARY)

clean:
	rm $(BINARY)

run: $(BINARY)
	INFLUX_URL=http://10.0.0.11:8086 ./$(BINARY)

test:
	go test ./...
	golint ./...
