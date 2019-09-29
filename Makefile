GOENV=GOOS=linux GOARCH=arm GOARM=6
GOCMD=go
GOBUILD=$(GOENV) $(GOCMD) build
GOCLEAN=$(GOCMD) clean

VERSION ?= $(shell git describe --tags --always --dirty 2> /dev/null )
LDFLAGS=-ldflags "-X=main.version=$(VERSION)"

spis=$(patsubst %.go, %, $(wildcard example/spi/*/*.go))
examples=$(patsubst %.go, %, $(wildcard example/*/*.go))
bins= $(spis) $(examples)

all: cmd/gppiio/gppiio $(bins)

cmd/gppiio/gppiio : $(wildcard cmd/gppiio/*.go)
	cd $(@D); \
	$(GOBUILD) $(LDFLAGS)

$(bins) : % : %.go
	cd $(@D); \
	$(GOBUILD)

clean: 
	$(GOCLEAN) ./...
