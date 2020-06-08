.PHONY: install

static/static_generated.go: $(filter-out static/static_generated.go,$(wildcard static/*))
	cd static && go generate

$(GOPATH)/bin/reserve: $(shell find . -name '*.go')
	go install github.com/s4y/reserve
	go install github.com/s4y/reserve/reserve

install: $(GOPATH)/bin/reserve
