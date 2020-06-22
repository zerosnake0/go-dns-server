.PHONY: image
image:
	docker build -t dns .

GOOS ?=
GOARCH ?=

bin/dns: main.go
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o $@ $<

clean:
	rm bin/dns