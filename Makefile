# goatee


run: build
	./goatee ${ARGS}

get:
	go get ./...

build:
	go build -ldflags="-s -w" -gcflags="-trimpath=${GOPATH}/src" -asmflags="-trimpath=${GOPATH}/src"

goinstall:
	go install

install: 
	cp ./goatee ${DESTDIR}

