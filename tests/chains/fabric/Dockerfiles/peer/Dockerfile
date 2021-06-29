ARG ALPINE_VER=3.13
ARG FABRIC_VER=2.2.0

FROM alpine:${ALPINE_VER} as builder
WORKDIR /root

ARG LEDGER_BACKUP_PATH

COPY ${LEDGER_BACKUP_PATH}/backup.tar ./backup.tar
RUN tar -xf backup.tar -C /root
RUN rm -rf backup.tar

FROM hyperledger/fabric-peer:${FABRIC_VER}

ARG PEER_CONFIG_FILE_PATH
ARG MSP_CONFIG_FILE_PATH
ARG EXTERNAL_BUILDER_BIN_PATH

COPY --from=builder /root/var/hyperledger/production /var/hyperledger/production
COPY ${PEER_CONFIG_FILE_PATH} /etc/hyperledger/fabric/core.yaml
COPY ${MSP_CONFIG_FILE_PATH} /etc/hyperledger/msp
COPY ${EXTERNAL_BUILDER_BIN_PATH} /builders/external/bin

EXPOSE 7051

CMD ["peer", "node", "start"]
