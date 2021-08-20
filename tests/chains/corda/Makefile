.PHONY: docker-images
docker-images:
	make -C ./yui-corda-ibc buildClientImage
	make -C ./yui-corda-ibc buildImage
	make -C ./yui-corda-ibc buildLightclientdImage

.PHONY: rebuild-chaincode-images
rebuild-chaincode-images: rebuild-chaincode1-image rebuild-chaincode2-image

.PHONY: rebuild-chaincode1-image
rebuild-chaincode1-image:
	docker build -f ./Dockerfiles/chaincode/Dockerfile -t fabric-chaincode-org1 \
		--build-arg LIGHTCLIENTD_IMAGE=yui-corda-ibc-lightclientd \
		--build-arg CHAINCODE_IMAGE=fabric-chaincode-org1 \
		.

.PHONY: rebuild-chaincode2-image
rebuild-chaincode2-image:
	docker build -f ./Dockerfiles/chaincode/Dockerfile -t fabric-chaincode-org2 \
		--build-arg LIGHTCLIENTD_IMAGE=yui-corda-ibc-lightclientd \
		--build-arg CHAINCODE_IMAGE=fabric-chaincode-org2 \
		.
