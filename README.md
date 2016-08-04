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
$ go get github.com/kayac/go-katsubushi
$ cd $GOPATH/github.com/kayac/go-katsubushi
make
```

## Usage

```
$ cd $GOPATH/github.com/kayac/go-katsubushi/cmd/katsubushi
./katsubushi -worker-id=1 -port=7238
./katsubushi -worker-id=1 -sock=/path/to/unix-domain.sock
```

## Protocol

katsubushi use protocol compatible with memcached (text only, not binary).

## Algorithm

katsubushi use algorithm like snowflake to generate ID.

## Commandline Options

### -worker-id

Required.
ID of the worker, must be unique in your service.

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

## Licence

[MIT](https://github.com/kayac/go-katsubushi/blob/master/LICENSE)

## Author

[handlename](https://github.com/handlename)
