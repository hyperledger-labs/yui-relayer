include ../../docker.mk

.PHONY: network
network:
	TAG=${DOCKER_TAG} $(DOCKER_COMPOSE) \
		-f ../docker-compose-test.yaml \
		up -d \
		tendermint-chain0 \
		ethereum-ganache0
	make wait-for-launch ATTEMPT=30 CONTAINER=ethereum-ganache0

.PHONY: test
test:
	./scripts/fixture
	./scripts/init-rly
	./scripts/handshake
	#./scripts/test-tx

.PHONY: network-down
network-down:
	TAG=${DOCKER_TAG} $(DOCKER_COMPOSE) \
		-f ../docker-compose-test.yaml \
		down -v --remove-orphans
