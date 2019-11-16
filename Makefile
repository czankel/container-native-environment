# Go parameters
GOCMD=go
GOYACC=goyacc
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get

.PHONY: cne test clean

cne:
	${GOBUILD} -o $@ -v

test:
	$(GOTEST) -v ./... -exec sudo

