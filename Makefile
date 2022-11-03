buildAll: buildLinux buildWindows buildMac

buildLinux:
	env GOOS=linux GOARCH=amd64 go build -o ./bin/yamlParser-linux-amd64 ./cmd/
	chmod +x ./bin/yamlParser-linux-amd64

buildWindows:
	env GOOS=windows GOARCH=amd64 go build -o ./bin/yamlParser-x64.exe ./cmd/

buildMac:
	env GOOS=darwin GOARCH=amd64 go build -o ./bin/yamlParser-darwin-amd64 ./cmd/
	env GOOS=darwin GOARCH=arm64 go build -o ./bin/yamlParser-darwin-arm64 ./cmd/

test:
	go test ./src/...

exampleStdout:
	./bin/yamlParser-linux-amd64 ./example.yml

example:
	./bin/yamlParser-linux-amd64 ./example.yml -f example.parsed.yml
