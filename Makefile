PROG 		:= $(shell go list -m)
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

.PHONY: clean

all: build

build:
	GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=$(CGO_ENABLED) go build -ldflags="$(LDFLAGS)" -o $(OUTPUT) ./cmd/helm-schema/

fmt:
	go fmt ./...

clean:
	rm -f $(PROG) $(PROG)-*

test/example: build
	./$(OUTPUT) test-chart/
