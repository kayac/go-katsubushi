# katsubushi

katsubushi(鰹節) is stand alone application to generate unique ID.

## Example

```
$ telnet localhost 11212
Trying ::1...
Connected to localhost.
Escape character is '^]'.
GET new
VALUE new 0 20
8070450532247928832
END
```

## Installation

Download from [releases](https://github.com/kayac/go-katsubushi/releases) or build from source code.

```
$ go get github.com/kayac/go-katsubushi/v2
$ cd $GOPATH/github.com/kayac/go-katsubushi
make
```

## Docker image

- [katsubushi/katsubushi](https://hub.docker.com/r/katsubushi/katsubushi/)
- [ghcr.io/kayac/go-katsubushi](https://github.com/kayac/go-katsubushi/pkgs/container/go-katsubushi)


```
$ docker pull katsubushi/katsubushi
$ docker run -p 11212:11212 katsubushi/katsubushi -worker-id 1
$ docker run -p 11212:11212 katsubushi/katsubushi -redis redis://your.redis.host:6379/0
```

## Usage

```
$ cd $GOPATH/github.com/kayac/go-katsubushi/cmd/katsubushi
./katsubushi -worker-id=1 -port=7238
./katsubushi -worker-id=1 -sock=/path/to/unix-domain.sock
```

## Protocol

katsubushi use protocol compatible with memcached.

Some commands are available with text and binary protocol.

But the others are available only with text protocol.

### API

#### GET, GETS

Binary protocol is also available only for single key GET.

```
GET id1 id2
VALUE id1 0 18
283890203179880448
VALUE id2 0 18
283890203179880449
END
```

VALUE(s) are unique IDs.

#### STATS

Returns a stats of katsubushi.

Binary protocol is also available.

```
STAT pid 8018
STAT uptime 17
STAT time 1487754986
STAT version 1.1.2
STAT curr_connections 1
STAT total_connections 2
STAT cmd_get 2
STAT get_hits 3
STAT get_misses 0
```

#### VERSION

Returns a version of katsubushi.

Binary protocol is available, too.

```
VERSION 1.1.2
```

#### QUIT

Disconnect an established connection.

## Protocol (HTTP)

katsubushi also runs an HTTP server specified with `-http-port`.

### GET /id

Get a single ID.

When `Accept` HTTP header is 'application/json', katsubushi will return an ID as JSON format as below.

```json
{"id":"1025441401866821632"}
```

Otherwise, katsubushi will return ID as text format.

```
1025441401866821632
```

### GET /ids?n=(number_of_ids)

Get multiple IDs.

`n` must be between 1 and 1000. Otherwise, katsubushi returns the 400 Bad Request error.

When `Accept` HTTP header is 'application/json', katsubushi will return an IDs as JSON format as below.

```json
{"ids":["1025442579472195584","1025442579472195585","1025442579472195586"]}
```

Otherwise, katsubushi will return ID as text format delimiterd with "\n".

```
1025442579472195584
1025442579472195585
1025442579472195586
```

### GET /stats

Returns a stats of katsubushi.

This API returns a JSON always.

```json
{
  "pid": 1859630,
  "uptime": 50,
  "time": 1664761614,
  "version": "1.8.0",
  "curr_connections": 1,
  "total_connections": 5,
  "cmd_get": 15,
  "get_hits": 25,
  "get_misses": 0
}
```

## Protocol (gRPC)

katsubushi also runs an gRPC server specified with `-grpc-port`.

See [grpc/README.md](grpc/README.md).

## Algorithm

katsubushi use algorithm like snowflake to generate ID.

Each ID is a 64 bit unsigned integer composed of the following bits.

```
 63                                           22 21           12 11           0
+-----------------------------------------------+---------------+-------------+
|              timestamp (42 bits)              | worker ID     | sequence    |
|                                               | (10 bits)     | (12 bits)   |
+-----------------------------------------------+---------------+-------------+
```

- **timestamp (42 bits)**: milliseconds elapsed since the katsubushi epoch (2015-01-01 00:00:00 UTC).
- **worker ID (10 bits)**: ID of the worker which generated the ID. Up to 1024 workers can run in a service.
- **sequence (12 bits)**: incremented for each ID generated in the same millisecond. Up to 4096 IDs can be generated per millisecond per worker.

### Limits

- **Lifetime**: The 42 bit timestamp can represent 2^42 milliseconds (about 139 years) from the epoch, so IDs can be generated until around the year 2154.
- **Throughput**: Each worker can generate up to 4096 IDs per millisecond, that is 4,096,000 IDs per second. With the maximum of 1024 workers, a service can generate up to about 4.2 billion IDs per second in total.

### JS-safe ID format

The default 64 bit IDs exceed `Number.MAX_SAFE_INTEGER` (2^53 - 1) in JavaScript, so they lose precision when handled as a number (e.g. parsed from JSON). When IDs are used as numeric database primary keys, they may be handled as numbers unintentionally, especially in dynamically typed languages.

To avoid this, katsubushi can generate IDs that fit within 2^53 - 1 with the `-js-safe-id` option.

```
 52                                    14 13       8 7           0
+----------------------------------------+----------+------------+
|  timestamp (39 bits, 10ms units)       | worker   | sequence   |
|                                        | (6 bits) | (8 bits)   |
+----------------------------------------+----------+------------+
```

- **timestamp (39 bits)**: 10 milliseconds units elapsed since the epoch (2015-01-01 00:00:00 UTC). 2^39 * 10ms is about 174 years, so IDs can be generated until around the year 2189.
- **worker ID (6 bits)**: up to 64 workers can run in a service.
- **sequence (8 bits)**: up to 256 IDs can be generated per 10 milliseconds per worker, that is 25,600 IDs per second.

Note:

- IDs in the JS-safe format are not compatible with IDs in the default format. Do not mix both formats in a service.
- When using the JS-safe format with `-redis`, use a different namespace (`?ns=`) from the default format because the worker ID ranges are different.
- The format of an ID can be determined by its value: IDs in the JS-safe format never exceed 2^53 - 1 (9007199254740991), while IDs in the default format always exceed it (a default format ID could be below 2^53 only within 25 days after the epoch, before katsubushi was released).

## Commandline Options

`-worker-id` or `-redis` is required.

All options can also be set via environment variables. For each option, the following environment variables are looked up in order, and the first one found is used. For example, for `-worker-id`:

1. `KATSUBUSHI_WORKER_ID`
2. `WORKER_ID`
3. `worker_id`

Commandline options take precedence over environment variables.

The `KATSUBUSHI_` prefixed form is recommended because non-prefixed names may conflict with unrelated environment variables (e.g. `PORT` or `VERSION` are commonly set by platforms).

### -worker-id

ID of the worker, must be unique in your service.

### -redis

URL of Redis server. e.g. `redis://example.com:6379/0`

`redis://{host}:{port}/{db}?ns={namespace}`

If you are using Redis Cluster, you will need to specify the URL as `rediscluster://{host}:{port}?ns={namespace}`.

This option is specified, katsubushi will assign an unique worker ID via Redis.

All katsubushi process for your service must use a same Redis URL.

### -min-worker-id -max-worker-id

These options work with `-redis`.

If we use multi katsubushi clusters, worker-id range for each clusters must not be overlapped. katsubushi can specify the worker-id range by these options.

### -port

Optional.
Port number of the memcached compatible server.
Default value is `11212`.
`0` means disable (e.g. to serve the HTTP or gRPC server only).
At least one of `-port`, `-sock`, `-http-port` or `-grpc-port` must be enabled.

### -sock

Optional.
Path of unix doamin socket.

### -idle-timeout

Optional.
Connection idle timeout in seconds.
`0` means infinite.
Default value is `600`.

### -log-level

Optional.
Default value is `info`.

### -log-format

Optional.
Log format, `text` or `json`.
Default value is `text`.

### -enable-pprof

Optional.
Boolean flag.
Enable profiling API by `net/http/pprof`.
Endpoint is `/debug/pprof`.

### -enable-stats

Optional.
Boolean flag.
Enable stats API by `github.com/fukata/golang-stats-api-handler`.
Endpoint is `/debug/stats`.

### -debug-port

Optional.
Port number for listen http used for `pprof` and `stats` API.
Defalut value is `8080`.

### -http-port

Optional.
Port number of HTTP server.
Default value is `0` (disabled).

### -grpc-port

Optional.
Port number of gRPC server.
Default value is `0` (disabled).

### -js-safe-id

Optional.
Boolean flag.
Generate IDs that fit within 2^53 - 1 (`Number.MAX_SAFE_INTEGER` in JavaScript).
See [JS-safe ID format](#js-safe-id-format) for details.


## Licence

[MIT](https://github.com/kayac/go-katsubushi/blob/master/LICENSE)

## Author

[handlename](https://github.com/handlename)
