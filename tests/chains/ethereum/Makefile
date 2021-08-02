include ../../docker.mk

SCRIPT_DIR ?= $(CURDIR)/scripts

NETWORK_ID0 ?= 12018
NETWORK_ID1 ?= 12019

export NETWORK_ID0
export NETWORK_ID1

.PHONY: docker-images
docker-images:
	make up-scaffold
	make deploy-contract
	make pause-scaffold
	make save-contract-address
	make docker-commit
	make down-scaffold

.PHONY: deploy-contract
deploy-contract:
	cd contract && npm run migrate

.PHONY: up-scaffold
up-scaffold:
	$(DOCKER_COMPOSE) up --build -d

.PHONY: down-scaffold
down-scaffold:
	$(DOCKER_COMPOSE) down --volumes --remove-orphans

.PHONY: pause-scaffold
pause-scaffold:
	$(DOCKER_COMPOSE) pause

.PHONY: unpause-scaffold
unpause-scaffold:
	$(DOCKER_COMPOSE) unpause

.PHONY: save-contract-address
save-contract-address:
	$(SCRIPT_DIR)/docker/saveContractAddress.sh $(NETWORK_ID0)
	$(SCRIPT_DIR)/docker/saveContractAddress.sh $(NETWORK_ID1)

.PHONY: docker-commit
docker-commit:
	$(SCRIPT_DIR)/docker/commitGanache.sh  $(DOCKER_REPO) $(DOCKER_TAG) ethereum-ganache0 ethereum-ganache0-scaffold $(NETWORK_ID0)
	$(SCRIPT_DIR)/docker/commitGanache.sh  $(DOCKER_REPO) $(DOCKER_TAG) ethereum-ganache1 ethereum-ganache1-scaffold $(NETWORK_ID1)
