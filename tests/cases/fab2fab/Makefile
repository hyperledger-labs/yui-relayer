include ../../docker.mk

.PHONY: network
network:
	TAG=${DOCKER_TAG} $(DOCKER_COMPOSE) \
		-f ../docker-compose-test.yaml \
		up -d \
		fabric-orderer.example.com fabric-data.example.com \
		fabric-peer0.org1.example.com fabric-chaincode.org1.example.com \
		fabric-peer0.org2.example.com fabric-chaincode.org2.example.com

.PHONY: test
test:
	@./scripts/fixture
	@./scripts/init-rly
	@./scripts/handshake
	@./scripts/test-tx

.PHONY: network-down
network-down:
	TAG=${DOCKER_TAG} $(DOCKER_COMPOSE) \
		-f ../docker-compose-test.yaml \
		down --volume --remove-orphans
