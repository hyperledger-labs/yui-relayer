include ../../docker.mk

FABRIC_VERSION    ?=2.2.0
FABRIC_CA_VERSION ?=1.4.7

FABRIC_BIN_DIR ?= $(CURDIR)/.bin
export PATH := $(FABRIC_BIN_DIR):$(PATH)

SCRIPT_DIR ?= $(CURDIR)/scripts

.PHONY: bin
bin:
	rm -rf $(CURDIR)/.bin
	curl -sSL https://bit.ly/2ysbOFE | bash -s -- $(FABRIC_VERSION) $(FABRIC_CA_VERSION) -d -s
	rm -rf $(CURDIR)/config
	mv $(CURDIR)/bin $(CURDIR)/.bin

.PHONY: clean
clean:
	rm -rf artifacts
	rm -rf backup
	rm -rf build
	rm -rf organizations

.PHONY: cryptogen
cryptogen:
	rm -rf $(CURDIR)/organizations
	$(FABRIC_BIN_DIR)/cryptogen generate --config $(CURDIR)/cryptogen/crypto-config.yaml --output ./organizations

.PHONY: configtx
configtx:
	rm -rf $(CURDIR)/artifacts
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
	rm -rf $(CURDIR)/build
	rm -rf $(CURDIR)/external-builders/config/**/tfspoc-paint/code.tar.gz
	$(SCRIPT_DIR)/deployCC.sh

.PHONY: backup
backup:
	rm -rf $(CURDIR)/backup
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
