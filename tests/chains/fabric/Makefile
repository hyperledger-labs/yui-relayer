include ../../docker.mk

FABRIC_VERSION    ?=2.2.0
FABRIC_CA_VERSION ?=1.4.7

FABRIC_BIN_DIR ?= $(CURDIR)/.bin
export PATH := $(FABRIC_BIN_DIR):$(PATH)

SCRIPT_DIR ?= $(CURDIR)/scripts

.PHONY: bin
bin:
	rm -rf ./.bin
	curl -sSL https://raw.githubusercontent.com/hyperledger/fabric/v$(FABRIC_VERSION)/scripts/bootstrap.sh | bash -s -- $(FABRIC_VERSION) $(FABRIC_CA_VERSION) -d -s
	rm -rf ./config
	mv ./bin ./.bin

.PHONY: clean
clean:
	rm -rf artifacts
	rm -rf backup
	rm -rf build
	rm -rf organizations
	rm -rf external-builders/config/**/fabibc/code.tar.gz

.PHONY: cryptogen
cryptogen:
	$(FABRIC_BIN_DIR)/cryptogen generate --config ./cryptogen/crypto-config.yaml --output ./organizations

.PHONY: configtx
configtx:
	$(SCRIPT_DIR)/configtx.sh

.PHONY: initial-data
initial-data:
	make cryptogen
	make configtx

.PHONY: up-scaffold
up-scaffold:
	IMAGE_TAG=$(FABRIC_VERSION) $(DOCKER_COMPOSE) up -d

.PHONY: pause-scaffold
pause-scaffold:
	IMAGE_TAG=$(FABRIC_VERSION) $(DOCKER_COMPOSE) pause

.PHONY: unpause-scaffold
unpause-scaffold:
	IMAGE_TAG=$(FABRIC_VERSION) $(DOCKER_COMPOSE) unpause

.PHONY: down-scaffold
down-scaffold:
	IMAGE_TAG=$(FABRIC_VERSION) $(DOCKER_COMPOSE) down --volumes --remove-orphans

.PHONY: channel
channel:
	$(SCRIPT_DIR)/createChannel.sh

.PHONY: chaincode
chaincode:
	$(SCRIPT_DIR)/deployCC.sh

.PHONY: backup
backup:
	$(SCRIPT_DIR)/docker/backupVolume.sh

.PHONY: snapshot
snapshot:
	make up-scaffold
	make channel
	make chaincode
	make pause-scaffold
	make backup
	make unpause-scaffold
	make down-scaffold

.PHONY: docker-images
docker-images:
	$(SCRIPT_DIR)/docker/buildImages.sh $(DOCKER_REPO) $(DOCKER_TAG)
