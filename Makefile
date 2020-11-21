GO       = go
VERSION != git rev-parse HEAD | cut -c 1-8
LDFLAGS  = -ldflags "-X main.VERSION=${VERSION}"

PHONY: all
all:
	go build ${LDFLAGS}

armv6:
	CC=arm-linux-gnueabi-gcc CGO_ENABLED=1 GOOS=linux GOARCH=arm GOARM=6 go build ${LDFLAGS}

install:
	go install ${LDFLAGS}

clean:
	rm bot
