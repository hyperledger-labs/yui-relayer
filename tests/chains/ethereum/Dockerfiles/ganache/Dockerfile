FROM trufflesuite/ganache-cli:v6.12.2

ARG CHAINID
ARG NETWORKID

ENV CHAINID ${CHAINID}
ENV NETWORKID ${NETWORKID}

WORKDIR /root

ADD mnemonic  ./mnemonic
ADD run.sh /run.sh

EXPOSE 8545

ENTRYPOINT ["/run.sh"]
