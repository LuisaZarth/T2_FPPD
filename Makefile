.PHONY: all build clean distclean

all: build

go.mod:
	go mod tidy

build: go.mod
	cd cliente && go build -o ../jogo

clean:
	rm -f jogo

distclean: clean
	rm -f go.mod go.sum
