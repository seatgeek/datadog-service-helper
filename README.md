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
    tags = ["dd-go-exprvar"]
}
```

## Current service backends

### php-fpm

- `PHP_FPM_CONFIG_FILE` (default: `/etc/dd-agent/conf.d/php_fpm.yaml`) path to the dd-agent `php_fpm.yaml` file.

Required service tag `dd-php-fpm`

### go_exprvar

- `GO_EXPR_TARGET_FILE` (default: `/etc/dd-agent/conf.d/go_expvar.yaml`) path to the dd-agent `go_expr.yaml` file.

Required service tag `dd-go-exprvar`

### redis

- `REDIS_TARGET_FILE` (default: `/etc/dd-agent/conf.d/redis.yaml`) path to the dd-agent `redis.yaml` file.

Required service tag `dd-redis`

## Local development

To get the dependencies and first build, please run:

```
make install
```

For easy Local development run

```
go install && \
    DONT_RELOAD_DATADOG=1 \
    TARGET_FILE_GO_EXPR=go_expr.yaml \
    TARGET_FILE_PHP_FPM=php_fpm.yaml \
    CONSUL_HTTP_ADDR=<consul client address>:8500 \
    datadog-fpm-monitor
```
