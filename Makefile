# Go parameters
GOCMD=go
GOYACC=goyacc
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get

VERSION = $(shell git describe --always --dirty)

DESTDIR ?= /usr/local/bin

.PHONY: cne test clean service/remote.proto

cne:
	${GOBUILD} -o $@ -v -ldflags="-X github.com/czankel/cne/config.CneVersion=${VERSION}"

cned:	cne
	ln -sf cne cned

install: cne
	sudo install -t ${DESTDIR} -o root -m a=rx,u+s $<

suid:	cne
	sudo chown root $<
	sudo chmod +s $<

test:
	$(GOTEST) -v ./... -exec sudo

proto: service/runtime.pb

service/runtime.pb: service/runtime.proto
	protoc --go_out=. --go_opt=paths=source_relative \
	--go-grpc_out=. --go-grpc_opt=paths=source_relative \
	service/runtime.proto


