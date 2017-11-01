# datadog-service-helper

The purpose of this tool is to easy datadog service monitoring in a docker environment where services change port and host all the time.

The daemon will run on each server you have (e.g. through Nomad) and tail the local consul agents service catalog, and write DataDog agent configuration files, based on the catalog state.

## Usage

If a consul service has a tag called `dd-<service_type>` the tool will ensure it will be monitored by the local DataDog agent.

Nomad example:

```hcl
service {
    name = "datadog-service-helper"
    port = "http"
    tags = ["dd-go-expvar"]
}
```

## Current service backends

### php-fpm

- `PHP_FPM_CONFIG_FILE` (default: `/etc/dd-agent/conf.d/php_fpm.yaml`) path to the dd-agent `php_fpm.yaml` file.

Required service tag `dd-php-fpm`

### go_expvar

- `GO_EXPVAR_CONFIG_FILE` (default: `/etc/dd-agent/conf.d/go_expvar.yaml`) path to the dd-agent `go_expvar.yaml` file.

Required service tag `dd-go-expvar`

### redis

- `REDIS_TARGET_FILE` (default: `/etc/dd-agent/conf.d/redis.yaml`) path to the dd-agent `redis.yaml` file.

Required service tag `dd-redis`

### TCP

- `TCP_CHECK_CONFIG_FILE` (default: `/etc/dd-agent/conf.d/tcp_check.yaml`) path to the dd-agent `tcp_check.yaml` file.

Required service tag `dd-tcp-check`

## Local development

To get the dependencies and first build, please run:

```
make install
```

For easy Local development run

```
go install && \
    DONT_RELOAD_DATADOG=1 \
    GO_EXPVAR_CONFIG_FILE=go_expvar.yaml \
    PHP_FPM_CONFIG_FILE=php_fpm.yaml \
    REDIS_TARGET_FILE=redis.yaml \
    CONSUL_HTTP_ADDR=<consul client address>:8500 \
    datadog-fpm-monitor
```
