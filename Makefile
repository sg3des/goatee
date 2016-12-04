

run: build
	./goatee

build:
	go build

install: 
	cp ./goatee /usr/bin/

