run-integration-test:
	docker-compose -f docker-compose.test.yaml up --build --force-recreate --exit-code-from integration
