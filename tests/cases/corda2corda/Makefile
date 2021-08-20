include ../../docker.mk

.PHONY: network
network:
	TAG=${DOCKER_TAG} $(DOCKER_COMPOSE) \
		-f ../docker-compose-test.yaml \
		up -d \
		yui-corda-ibc1 \
		yui-corda-ibc2
	while ! wget -q -O - 'localhost:9999'; do sleep 1; done
	while ! wget -q -O - 'localhost:9998'; do sleep 1; done
	while ! wget -q -O - 'localhost:29999'; do sleep 1; done
	while ! wget -q -O - 'localhost:29998'; do sleep 1; done

.PHONY: test
test:
	./scripts/init-rly
	./scripts/handshake
	./scripts/test-tx

.PHONY: network-down
network-down:
	TAG=${DOCKER_TAG} $(DOCKER_COMPOSE) \
		-f ../docker-compose-test.yaml \
		down --volume --remove-orphans
