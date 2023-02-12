run-integration-test:
	docker-compose -f docker-compose.test.yaml up --build --force-recreate --exit-code-from integration

build:
	go build -ldflags "-s -w" -o bazels3cache ./...

build-debug:
	go build -tags debug -o bazels3cache ./...

start:
	./bazels3cache --bucket bazel

start-debug:
	./bazels3cache --bucket bazel --s3url http://localhost:9444/s3

stop:
	./bazels3cache -stop
