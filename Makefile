# Go parameters
GOCMD=go
GOYACC=goyacc
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get

VERSION = $(shell git describe --always --dirty)

DESTDIR ?= /usr/local/bin

.PHONY: cne test clean

cne:
	${GOBUILD} -o $@ -v -ldflags="-X github.com/czankel/cne/config.CneVersion=${VERSION}"

install: cne
	sudo install -t ${DESTDIR} -o root -m a=rx,u+s $<

suid:	cne
	sudo chown root $<
	sudo chmod +s $<


test:
	$(GOTEST) -v ./... -exec sudo
