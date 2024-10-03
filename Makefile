default: build

test:
	go test ./...

build:
	go build -o tflint-ruleset-uw-kafka-config

install: build
	mkdir -p ~/.tflint.d/plugins
	mv ./tflint-ruleset-uw-kafka-config ~/.tflint.d/plugins

lint:
	pre-commit run --all-files
