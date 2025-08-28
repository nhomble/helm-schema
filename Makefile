PROG 		:= $(shell go list -m)
PLUGIN      := schema
LDFLAGS		:= -w -s
GOOS		?= $(shell go env GOOS)
GOARCH		?= $(shell go env GOARCH)
CGO_ENABLED	?= 0

ifeq ($(GOOS),windows)
    OUTPUT = $(PROG)-$(GOOS)-$(GOARCH).exe
else
    OUTPUT = $(PROG)-$(GOOS)-$(GOARCH)
endif

ifeq ($(GOOS),$(shell go env GOOS))
ifeq ($(GOARCH),$(shell go env GOARCH))
    OUTPUT = $(PROG)
endif
endif

.PHONY: clean test fmt test/example plugin/uninstall

# install is not idempotent
.IGNORE: plugin/install plugin/uninstall

all: build plugin/install

build:
	GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=$(CGO_ENABLED) go build -ldflags="$(LDFLAGS)" -o $(OUTPUT) ./cmd/helm-schema/

fmt:
	go fmt ./...

test:
	go test ./pkg/...

test/example: build
	./$(OUTPUT) test-charts/basic/

plugin/install: build
	helm plugin install . 

plugin/uninstall:
	 helm plugin uninstall $(PLUGIN)

clean: plugin/uninstall
	rm -f $(PROG) $(PROG)-*
