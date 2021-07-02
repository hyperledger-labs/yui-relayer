DOCKER         ?= docker
DOCKER_COMPOSE ?= docker-compose
DOCKER_REPO    ?= ""
DOCKER_TAG     ?= latest
DOCKER_BUILD   ?= $(DOCKER) build --rm --no-cache --pull
