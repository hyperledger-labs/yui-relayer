ARG GO_VER=1.15.8
ARG ALPINE_VER=3.13

FROM golang:${GO_VER}-alpine${ALPINE_VER} as builder

COPY ./Dockerfiles/chaincode/env.sh /root/env.sh

ARG CHAINCODE_CCID
ARG CHAINCODE_ADDRESS
RUN echo export CHAINCODE_CCID=${CHAINCODE_CCID} >> /root/env.sh
RUN echo export CHAINCODE_ADDRESS=${CHAINCODE_ADDRESS} >> /root/env.sh

RUN apk add --no-cache gcc git libc-dev tzdata

COPY ./chaincode/fabibc /root/chaincode/fabibc

WORKDIR /root/chaincode/fabibc

RUN go build -v -mod readonly -o fabibc .

FROM alpine:${ALPINE_VER}

COPY --from=builder /root/chaincode/fabibc/fabibc /usr/bin
COPY --from=builder /root/env.sh /etc/profile.d/env.sh

CMD ["/usr/bin/fabibc"]
