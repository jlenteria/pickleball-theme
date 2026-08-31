plugin: export CGO_ENABLED=1

plugin:
	go build -buildmode=plugin -trimpath -o plugin.so ./main.go

test:
	go test ./...

test-mono:
	go test -tags mono ./...

vet:
	go vet ./...

check: test test-mono vet
