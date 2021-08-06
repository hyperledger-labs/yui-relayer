include ../../docker.mk

SCRIPT_DIR ?= $(CURDIR)/scripts

NETWORK_ID0 ?= 13370
NETWORK_ID1 ?= 13380

export NETWORK_ID0
export NETWORK_ID1

.PHONY: docker-images
docker-images:
	make up-scaffold
	make deploy-contract
	make save-contract-address
	make docker-commit
	make down-scaffold

.PHONY: deploy-contract
deploy-contract:
	make wait-for-launch ATTEMPT=30 CONTAINER=ethereum-ganache0-scaffold
	cd contract && npm run migrate

.PHONY: up-scaffold
up-scaffold:
	$(DOCKER_COMPOSE) up --build -d

.PHONY: down-scaffold
down-scaffold:
	$(DOCKER_COMPOSE) down --volumes --remove-orphans

.PHONY: save-contract-address
save-contract-address:
	$(SCRIPT_DIR)/docker/saveContractAddress.sh $(NETWORK_ID0)
	$(SCRIPT_DIR)/docker/saveContractAddress.sh $(NETWORK_ID1)

.PHONY: docker-commit
docker-commit:
	$(SCRIPT_DIR)/docker/commitImage.sh  $(DOCKER_REPO) $(DOCKER_TAG) ethereum-ganache0 ethereum-ganache0-scaffold $(NETWORK_ID0)
	$(SCRIPT_DIR)/docker/commitImage.sh  $(DOCKER_REPO) $(DOCKER_TAG) ethereum-ganache1 ethereum-ganache1-scaffold $(NETWORK_ID1)
