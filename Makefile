build:
	go build -o bazels3cache ./...

build-debug:
	go build -tags debug -o bazels3cache ./...

start:
	./bazels3cache --bucket bazel

start-debug:
	./bazels3cache --bucket bazel --s3url http://localhost:9444/s3

stop:
	./bazels3cache -stop

run-integration-test:
	docker-compose -f test/docker-compose.test.yaml up --build --exit-code-from integration --force-recreate

run-benchmark:
	docker-compose -f test/docker-compose.benchmark.yaml --env-file ./test/benchmark.env up --build --force-recreate --exit-code-from integration

run-benchmark-no-recreate:
	docker-compose -f test/docker-compose.benchmark.yaml --env-file ./test/benchmark.env up --build --exit-code-from integration
