include ../../docker.mk

.PHONY: network
network:
	TAG=${DOCKER_TAG} $(DOCKER_COMPOSE) \
		-f ../docker-compose-test.yaml \
		up -d \
		tendermint-chain0 tendermint-chain1

.PHONY: test
test:
	./scripts/fixture
	./scripts/init-rly
	./scripts/handshake
	./scripts/test-channel-upgrade
	./scripts/test-create-client-success-single
	./scripts/test-create-client-fail-already-created
	./scripts/test-create-client-fail-unexist
	./scripts/test-create-connection-fail-already-created
	./scripts/test-create-connection-fail-unexist
	./scripts/test-create-channel-fail-already-created
	./scripts/test-create-channel-fail-unexist
	./scripts/test-tx
	./scripts/test-service

.PHONY: network-down
network-down:
	TAG=${DOCKER_TAG} $(DOCKER_COMPOSE) \
		-f ../docker-compose-test.yaml \
		down --volumes --remove-orphans
