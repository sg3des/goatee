

run: build
	./goatee ${ARGS}

build:
	go build

goinstall:
	go install

install: 
	cp ./goatee /usr/bin/

