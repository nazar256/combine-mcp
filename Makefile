.PHONY: build test clean run example

build:
	go build -o combine-mcp ./cmd

test:
	go test ./...

clean:
	rm -f combine-mcp
	rm -f examples/test-server/test-server
	rm -f examples/test_client

run: build
	./combine-mcp

example: build
	go build -o examples/test-server/test-server ./examples/test-server
	go build -o examples/test_client examples/test_client.go
	export MCP_CONFIG=./examples/config.json && ./examples/test_client
