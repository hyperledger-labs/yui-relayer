## Prerequisites

```shell script
# download fabric binaries
$ make bin
```

## Usage

### create Snapshot docker image
```shell
$ make initial-data // if reset network config
$ make snapshot
$ make docker-images
$ docker-compose up -d
```

### deploy Chaincode Image to Docker Registry(TODO)
```
