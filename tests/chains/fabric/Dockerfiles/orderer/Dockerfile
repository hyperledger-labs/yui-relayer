ARG ALPINE_VER=3.13
ARG FABRIC_VER=2.2.0

FROM alpine:${ALPINE_VER} as builder
WORKDIR /root

ARG LEDGER_BACKUP_PATH

COPY ${LEDGER_BACKUP_PATH}/backup.tar ./backup.tar
RUN tar -xf backup.tar -C /root
RUN rm -rf backup.tar

FROM hyperledger/fabric-orderer:${FABRIC_VER}

ARG ORDERER_CONFIG_FILE_PATH
ARG MSP_CONFIG_FILE_PATH
ARG ORDERER_BLOCK_FILE_PATH

COPY --from=builder /root/var/hyperledger/production /var/hyperledger/production
COPY ${ORDERER_CONFIG_FILE_PATH} /var/hyperledger/orderer/orderer.yaml
COPY ${MSP_CONFIG_FILE_PATH} /etc/hyperledger/msp
COPY ${ORDERER_BLOCK_FILE_PATH} /etc/hyperledger/fabric/orderer.block

EXPOSE 7050

CMD orderer
