GO      = go
VERSION = $(shell git rev-parse HEAD | cut -c 1-8)
LDFLAGS = -ldflags "-X main.VERSION=${VERSION}"

PHONY: all
all:
	go build ${LDFLAGS}

install:
	go install ${LDFLAGS}

clean:
	rm bot
