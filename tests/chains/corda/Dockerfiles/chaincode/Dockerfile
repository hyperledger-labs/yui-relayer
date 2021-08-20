ARG LIGHTCLIENTD_IMAGE
ARG CHAINCODE_IMAGE

FROM $CHAINCODE_IMAGE AS builder

FROM $LIGHTCLIENTD_IMAGE
COPY --from=builder /usr/bin/fabibc /usr/bin/fabibc-chaincode
COPY --from=builder /etc/profile.d/env.sh /etc/profile.d/env.sh
COPY ./Dockerfiles/chaincode/fabibc /usr/bin/fabibc

RUN apt-get update && apt-get install -y netcat

CMD ["/usr/bin/fabibc"]
