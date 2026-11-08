# make qa      the gate: gofmt, vet, the tests under the race detector, the static build
# make deploy  the gate, then deploy/install.sh onto the kourt box
# Run these on the devbox, not on the Mac (README.md).
BOX ?= root@138.68.5.120

.PHONY: qa build deploy

qa:
	@out=$$(gofmt -l .); if [ -n "$$out" ]; then echo "gofmt wants: $$out"; exit 1; fi
	go vet ./...
	go test -race -count=1 ./...
	CGO_ENABLED=0 go build -trimpath -o bin/ephesus .

build:
	CGO_ENABLED=0 go build -trimpath -o bin/ephesus .

deploy: qa
	deploy/install.sh $(BOX)
