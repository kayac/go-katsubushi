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

## Commandline Options

`-worker-id` or `-redis` is required.

### -worker-id

ID of the worker, must be unique in your service.

### -redis

URL of Redis server. e.g. `redis://example.com:6379/0`

`redis://{host}:{port}/{db}?ns={namespace}`

This option is specified, katsubushi will assign an unique worker ID via Redis.

All katsubushi process for your service must use a same Redis URL.

### -min-worker-id -max-worker-id

These options work with `-redis`.

If we use multi katsubushi clusters, worker-id range for each clusters must not be overlapped. katsubushi can specify the worker-id range by these options.

### -port

Optional.
Port number used for connection.
Default value is `11212`.

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


## Licence

[MIT](https://github.com/kayac/go-katsubushi/blob/master/LICENSE)

## Author

[handlename](https://github.com/handlename)
