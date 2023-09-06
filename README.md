# YALB (Yet Another Load Balancer) Documentation

## Installation

- Build from source using `go build -v .` from the root dir.

## Configuration

- Config file is written in yaml
- Have `YALB_CONFIG` environment variable that should be a path to your `.yaml` config
- An example config can be [here](./yalb.yaml)

## Features

Common Load-balancing algorithms:

- `round-robin`

- `least-connections`

Healthcheck support

- Add an endpoint say `/ping` that returns `200 OK` on `GET`

## Usage

Get the binary using the instructions [here](#installation):
 - ```sh
   export YALB_CONFIG=/path/to/your/configfile.yaml && ./yalb
   ```
