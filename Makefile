

run: build
	./goatee ${ARGS}

build:
	go build

# goinstall:
# 	go install

install: 
	go install
	sudo cp hex.lang /usr/share/gtksourceview-2.0/language-specs/
	# cp ./goatee /usr/bin/

