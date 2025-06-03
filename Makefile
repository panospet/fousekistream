all: build-simple build-radio

.PHONY: build-simple
build-simple:
	CGO_ENABLED=0 go build -ldflags='-w -s -extldflags "-static"' -o ./bin/simple-stream ./cmd/simple-stream/main.go

.PHONY: build-radio
build-radio:
	CGO_ENABLED=0 go build -ldflags='-w -s -extldflags "-static"' -o ./bin/radio-style-stream ./cmd/radio-style-stream/main.go

.PHONY: container
container: ## create docker container
	docker build -t registry.panos.pet/fousekistream .

.PHONY: container-push
container-push: container ## push docker image to registry
	docker push registry.panos.pet/fousekistream

.PHONY: run
run: ## run the application
	go run main.go
