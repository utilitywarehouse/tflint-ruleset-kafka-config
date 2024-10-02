default: build

test:
	go test ./...

build:
	go build

install: build
	mkdir -p ~/.tflint.d/plugins
	mv ./tflint-ruleset-kafka-config ~/.tflint.d/plugins

lint:
	pre-commit run --all-files
