buildAll: buildLinux buildWindows buildMac

buildLinux:
	env GOOS=linux GOARCH=amd64 go build -o ./bin/zparser-linux-amd64 ./cmd/
	chmod +x ./bin/zparser-linux-amd64

buildWindows:
	env GOOS=windows GOARCH=amd64 go build -o ./bin/zparser-x64.exe ./cmd/

buildMac:
	env GOOS=darwin GOARCH=amd64 go build -o ./bin/zparser-darwin-amd64 ./cmd/
	env GOOS=darwin GOARCH=arm64 go build -o ./bin/zparser-darwin-arm64 ./cmd/

installLint:
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.50.1

fmt:
	gofmt -s -w ./src/. ./cmd/.

lint:
	golangci-lint run ./src/... ./cmd/... --verbose

lintFix:
	golangci-lint run ./src/... ./cmd/... --verbose --fix

test:
	go test ./src/...

exampleStdout:
	./bin/zparser-linux-amd64 ./example.yml

example:
	./bin/zparser-linux-amd64 ./example.yml -f example.parsed.yml
