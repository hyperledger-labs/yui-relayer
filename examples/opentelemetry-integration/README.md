# OpenTelmetry integration example

This is an example of OpenTelmetry integration using the test cases in this repository.
The telemetry data collected by OpenTelmetry can be visualized in Grafana, which is also included in this example.

## Usage

### Setup

First, build `yrly` and docker images:

```sh
make -C ../.. build
make -C ../../tests/chains/tendermint docker-images
```

Next, start the containers for OpenTelemetry and the networks, and initialize the relayer:

```sh
docker compose up -d
make -C ../../tests/cases/tm2tm network
../../tests/cases/tm2tm/scripts/fixture
../../tests/cases/tm2tm/scripts/init-rly
```

Once the setup is complete, Grafana will eb available at http://localhost:3000.

### Run relayer

To send telemetry data to the OpenTelemetry Collector, you first need to set the following environment variables because the test scripts set `OTEL_*_EXPORTER` to `console` if they are not set:

```
export YRLY_ENABLE_TELEMETRY=true
export OTEL_TRACES_EXPORTER=otlp
export OTEL_METRICS_EXPORTER=otlp
export OTEL_LOGS_EXPORTER=otlp
export OTEL_EXPORTER_OTLP_INSECURE=true
```

After setting the environment, run the handshake script to create a client and establish a connection and channel:

```sh
../../tests/cases/tm2tm/scripts/handshake
```

Once the channel is established, you can run any of the test scripts prefixed `test-` to send telemetry data.
For example, the following command run `yrly service start` for a while:

```sh
../../tests/cases/tm2tm/scripts/test-service
```

To check the telemetry data, access http://localhost:3000.


### Clean up

Run the following commands to remove all the containers:

```
docker compose down
make -C ../../tests/cases/tm2tm network-down
```
