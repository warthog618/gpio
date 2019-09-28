GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean

VERSION ?= $(shell git describe --tags --always --dirty 2> /dev/null )
LDFLAGS=-ldflags "-X=main.version=$(VERSION)"

srcs=$(wildcard cmd/gppiio/gppiio*.go)


cmd/gppiio/gppiio : $(srcs)
	cd $(@D); \
	GOOS=linux GOARCH=arm GOARM=6 $(GOBUILD) $(LDFLAGS)

clean: 
	$(GOCLEAN) ./...


