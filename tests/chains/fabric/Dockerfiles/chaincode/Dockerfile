ARG GO_VER=1.15.8

FROM golang:${GO_VER}-buster as builder

COPY ./Dockerfiles/chaincode/env.sh /root/env.sh

ARG CHAINCODE_CCID
ARG CHAINCODE_ADDRESS
RUN echo export CHAINCODE_CCID=${CHAINCODE_CCID} >> /root/env.sh
RUN echo export CHAINCODE_ADDRESS=${CHAINCODE_ADDRESS} >> /root/env.sh

COPY ./chaincode/fabibc /root/chaincode/fabibc

WORKDIR /root/chaincode/fabibc

RUN go build -v -mod readonly -o fabibc .

FROM debian:buster

COPY --from=builder /root/chaincode/fabibc/fabibc /usr/bin
COPY --from=builder /root/env.sh /etc/profile.d/env.sh

CMD ["/usr/bin/fabibc"]
