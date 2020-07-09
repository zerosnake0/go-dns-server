GOOS ?=
GOARCH ?=

bin/dns: main.go
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o $@ $<

.PHONY: image
image:
	docker build -t dns .

.PHONY: clean
clean:
	rm bin/dns