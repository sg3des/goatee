
run: build
	./goatee ${ARGS}

get:
	go get ./...

build:
	go build

goinstall:
	go install

install: 
	cp ./goatee ${DESTDIR}

