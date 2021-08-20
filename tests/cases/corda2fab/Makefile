include ../../docker.mk

.PHONY: network
network:
	TAG=${DOCKER_TAG} $(DOCKER_COMPOSE) \
		-f ../docker-compose-test.yaml \
		up -d \
		yui-corda-ibc1 \
		fabric-orderer.example.com fabric-data.example.com \
		fabric-peer0.org1.example.com fabric-chaincode.org1.example.com
	while ! wget -q -O - 'localhost:9999'; do sleep 1; done
	while ! wget -q -O - 'localhost:29999'; do sleep 1; done
	while ! wget -q -O - 'localhost:17051'; do sleep 1; done

.PHONY: test
test:
	./scripts/fixture
	./scripts/init-rly
	./scripts/handshake
	./scripts/test-tx

.PHONY: network-down
network-down:
	TAG=${DOCKER_TAG} $(DOCKER_COMPOSE) \
		-f ../docker-compose-test.yaml \
		down --volume --remove-orphans
