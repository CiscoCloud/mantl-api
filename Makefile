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

build: deps
	@mkdir -p bin/
	go build -o bin/$(NAME)

test: deps
	go test $(TEST) $(TESTARGS) -timeout=30s -parallel=4
	go vet $(TEST)

quicktest:
	go test $(TEST) $(TESTARGS) -timeout=30s -parallel=4

docker: deps
	docker build -t $(DOCKERREPO)/$(NAME) .
	docker tag -f $(DOCKERREPO)/$(NAME) $(DOCKERREPO)/$(NAME):$(VERSION)

dockerpush: docker
	docker push $(DOCKERREPO)/$(NAME):$(VERSION)
