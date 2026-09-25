# codex-netmon

`codex-netmon` is a macOS command-line tool that measures Wi-Fi traffic attributed to Codex processes and commands launched by Codex. It exports OpenTelemetry metrics over OTLP/HTTP, so you can chart the traffic in Grafana or another OpenTelemetry-compatible system.

## Requirements

- macOS with the built-in `nettop` and `ps` commands
- An OTLP/HTTP metrics receiver, such as OpenTelemetry Collector
- Go 1.26.5 to build from source; [mise](https://mise.jdx.dev/) manages the development toolchain

## Install

```sh
go install github.com/yudai2929/codex-netmon@latest
```

Alternatively, download the macOS arm64 or amd64 archive from [GitHub Releases](https://github.com/yudai2929/codex-netmon/releases). If the `go install` command is not on your `PATH`, add `$(go env GOBIN)` when it is set, or `$(go env GOPATH)/bin` otherwise. To build from a checkout instead:

```sh
mise install
mise run build
./dist/codex-netmon -help
```

## Run

Start your OTLP/HTTP receiver first, then run:

```sh
codex-netmon -endpoint http://127.0.0.1:4318/v1/metrics -interval 5s
```

The endpoint above is the default. `-interval` controls both sampling and metric export and defaults to `5s`. Stop the foreground process with Ctrl-C. The tool runs only on macOS and needs access to the operating system's process and network information. If macOS prompts for permissions, grant only those required by your setup.

For an optional per-user background service from a source checkout, run `mise run service-start`; use `mise run service-stop` to stop it. This LaunchAgent uses the default local endpoint and writes logs under `.run/` in the checkout.

## Metrics

| OTLP metric | Type | Unit | Meaning |
| --- | --- | --- | --- |
| `codex.wifi.received` | cumulative counter | bytes | Wi-Fi bytes received |
| `codex.wifi.sent` | cumulative counter | bytes | Wi-Fi bytes sent |

Each metric includes `process` (the executable name) and `process_kind` (`codex` or `command`). `codex` covers recognized Codex desktop, service, renderer, and CLI processes. `command` covers descendants launched by Codex command hosts. The metric resource has `service.name=codex-netmon`. Prometheus commonly exposes the counters as `codex_wifi_received_bytes_total` and `codex_wifi_sent_bytes_total`.

For a Grafana time series of traffic over the selected interval, use `sum(increase(codex_wifi_received_bytes_total[$__rate_interval]))` and the matching `sent` expression. For a total over the selected time range, use `sum(increase(codex_wifi_received_bytes_total[$__range])) + sum(increase(codex_wifi_sent_bytes_total[$__range]))`. Use `process_kind="command"` to isolate commands launched by Codex.

## How attribution works

At each sample, the tool reads macOS `nettop -t wifi` counters and a `ps` process tree. It matches known Codex executables and their command descendants, then exports byte deltas between samples. A newly observed process is counted from its first visible `nettop` value only if it was absent from the previous process tree.

This is an estimate of traffic associated with Codex processes, not a packet-level accounting or an OpenAI API usage meter. Very short-lived processes may finish between samples. Traffic shifted into a separate daemon, container, VPN interface, or another network interface may be absent. Other network activity by a matched process is included. Wi-Fi link overhead is not included. Measurements start when the tool starts; past traffic cannot be recovered.

## Development

```sh
mise install
mise run test
mise run build
```

This project is licensed under the [MIT License](LICENSE).
