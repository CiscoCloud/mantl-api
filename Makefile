NAME = $(shell awk -F\" '/^const Name/ { print $$2 }' main.go)
VERSION = $(shell awk -F\" '/^const Version/ { print $$2 }' main.go)
TESTDEPS = $(shell go list -f '{{range .TestImports}}{{.}} {{end}}' $(shell glide novendor))
DOCKERREPO=ciscocloud

all: deps build

deps:
	glide install
	echo $(TESTDEPS) | xargs -n1 go get

updatedeps:
	glide update

quickbuild:
	@mkdir -p bin/
	go build -o bin/$(NAME)

build: deps quickbuild

quicktest:
	go test $(shell glide novendor) -timeout=30s -parallel=4

test: deps quicktest
	go vet $(shell glide novendor)

docker: deps
	find . -name ".DS_Store" -depth -exec rm {} \;
	docker build -t $(DOCKERREPO)/$(NAME) .
	docker tag -f $(DOCKERREPO)/$(NAME) $(DOCKERREPO)/$(NAME):$(VERSION)

pushedge: docker
	docker tag -f $(DOCKERREPO)/$(NAME) $(DOCKERREPO)/$(NAME):edge
	docker push $(DOCKERREPO)/$(NAME):edge

push: docker
	docker push $(DOCKERREPO)/$(NAME):$(VERSION)
	docker push $(DOCKERREPO)/$(NAME):latest
