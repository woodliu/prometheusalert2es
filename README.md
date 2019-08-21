## Prometheus Alerts To Elasticsearch

### Prepare

`golang version: 1.12.9`

`prmetheus version: 2.3.1`

`elasticsearch version:  6.8`

### Build

`go build -o  prometheusalert2es main.go`

### Usage

```shell
prometheusalert2es --esurl=${url} --esusername=${username} --espasswd=${passwd}
```

`prometheusalert2es` default listen on port 8888, you can specific by command parameter `--port`

#### Config prometheus

Configure in prometheus.yml，add `prometheusalert2es`  target `10.10.10.2:8888`

```yaml
alerting:
  alertmanagers:
    - static_configs:
      - targets:
        - 10.10.10.2:8888
```

#### Config elasticsearch

Make sure elasticsearch URI's scheme is `https`, and you have the `username` and `password` for elasticsearch basic authentication.

