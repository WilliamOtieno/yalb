# YALB (Yet Another Load Balancer) Documentation

## Installation

- Build from source using `go build -v .` from the root dir.

## Configuration

- Config file is written in yaml
- Have `YALB_CONFIG` environment variable that should be a path to your `.yaml` config
- An example config can be [here](./yalb.yaml)

## Usage

- Load-balancing algorithms currently supported are
    `round-robin`
    `least-connections`
