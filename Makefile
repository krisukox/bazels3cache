run-integration-test:
	docker-compose -f docker-compose.test.yaml up --build --force-recreate --exit-code-from integration

build:
	go build -o bazels3cache app/main.go

start:
	./bazels3cache -bucket bazel -s3url http://localhost:9444/s3

stop:
	./bazels3cache -stop
