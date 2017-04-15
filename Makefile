# goatee


run: build
	./goatee ${ARGS}

get:
	go get ./...

build:
	go build -ldflags="-s -w"

goinstall:
	go install

install: 
	cp ./goatee ${DESTDIR}

