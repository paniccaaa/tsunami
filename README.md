# Tsunami

Tsunami is an HTTP load testing tool for stress testing web services with a constant request rate. It features a CLI for terminal usage and a web UI for visual monitoring


## Install

### Pre-compiled binaries

Download from [GitHub Releases](https://github.com/paniccaaa/tsunami/releases).

### Homebrew (macOS/Linux)

```sh
brew install paniccaaa/tap/tsunami
```

### Source

```sh
git clone https://github.com/paniccaaa/tsunami.git
cd tsunami
make build
```

Requires Go 1.24+ and Node.js 20+.

### Go (CLI-only version without web UI!)

```sh
go install github.com/paniccaaa/tsunami@latest
```

### Docker

```sh
docker-compose up --build
# Open http://localhost:8080
```

## Usage manual

```
Usage: tsunami [global flags] <command> [command flags]

Global flags:
  --cpus int    Number of CPUs to use (default: all)
  -h, --help    Show help
  -v, --version Print version

Commands:
  attack      Run a load test
  serve       Start web UI server
```

### `tsunami attack`

```
Usage: tsunami attack [flags]

Flags:
  -u, --url string             Target URL to attack. Required.
  -m, --method string          HTTP method. (default "GET")
  -b, --body string            Request body.
  -H, --headers stringArray    Request headers. Can be repeated.
  -r, --rate string            Request rate. (default "100/1s")
  -d, --duration duration      Attack duration. 0 means infinite. (default 0)
  -t, --timeout duration       Request timeout. (default 10s)
  -w, --workers uint           Number of workers. (default 50)
  -c, --connections uint       Number of connections. (default 100)
  -o, --output string          Output file for JSON report. (default "stdout")
```

#### `-r, --rate`

Specifies the request rate in `N/Vunit` format where:
- `N` is the number of requests
- `V` is the time value
- `unit` is one of: `ms`, `s`, `m`, `h`

```
100/1s    100 requests per second
50/500ms  100 requests per second
1000/1m   ~16.6 requests per second
```

#### `-d, --duration`

Attack duration. When set to `0`, the attack runs until interrupted with `Ctrl+C`.

```
30s       30 seconds
5m        5 minutes
1h        1 hour
0         infinite (until Ctrl+C)
```

#### `-o, --output`

Output destination. Use `stdout` for terminal text output or a file path for JSON report.

```sh
# Text output to terminal
tsunami attack -u https://example.com

# JSON output to file
tsunami attack -u https://example.com -o results.json
```

### `tsunami serve`

```
Usage: tsunami serve [flags]

Flags:
  -p, --port int    Server port. (default 8080)
```

Starts an HTTP server with web UI and REST API.

```sh
tsunami serve            
2026/01/24 16:28:40 Starting Tsunami HTTP server on http://localhost:8080
```

![ui](image.png)

#### API endpoints

```
POST /api/attack/start           Start a load test
POST /api/attack/stop            Stop current test
GET  /api/attack/status          Get status and live metrics
GET  /api/attack/results         Get final results
GET  /api/attack/results/download Download results as JSON
WS   /ws/metrics                 Real-time metrics stream
```

## Examples

### Basic attack

```sh
tsunami attack -u https://httpbin.org/get
```

### POST request with JSON body

```sh
tsunami attack \
  -u https://httpbin.org/post \
  -m POST \
  -H "Content-Type: application/json" \
  -b '{"name": "test"}'
```

### High-rate attack with duration

```sh
tsunami attack \
  -u https://example.com \
  -r 1000/1s \
  -d 1m \
  -w 100 \
  -c 200
```

### Save results

```sh
tsunami attack \
  -u https://example.com \
  -d 30s \
  -o report.json
```

## License

MIT
