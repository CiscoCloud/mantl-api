TEST?=./...
NAME = $(shell awk -F\" '/^const Name/ { print $$2 }' main.go)
VERSION = $(shell awk -F\" '/^const Version/ { print $$2 }' main.go)
DEPS = $(shell go list -f '{{range .TestImports}}{{.}} {{end}}' ./...)
DOCKERREPO=ciscocloud

all: deps build

deps:
	go get -d -v ./...
	echo $(DEPS) | xargs -n1 go get -d

updatedeps:
	go get -u -v ./...
	echo $(DEPS) | xargs -n1 go get -d

quickbuild:
	go build -o bin/$(NAME)

build: deps quickbuild

quicktest:
	go test $(TEST) $(TESTARGS) -timeout=30s -parallel=4

test: deps quicktest
	go vet $(TEST)

docker: deps
	docker build -t $(DOCKERREPO)/$(NAME) .
	docker tag -f $(DOCKERREPO)/$(NAME) $(DOCKERREPO)/$(NAME):$(VERSION)

pushedge: docker
	docker tag -f $(DOCKERREPO)/$(NAME) $(DOCKERREPO)/$(NAME):edge
	docker push $(DOCKERREPO)/$(NAME):edge

push: docker
	docker push $(DOCKERREPO)/$(NAME):$(VERSION)
	docker push $(DOCKERREPO)/$(NAME):latest
