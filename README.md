# etcdv3-browser

A simple etcd (v3) web-based browser.

![etcd browser](https://rustyx.org/temp/etcdv3-browser.png)

Main features:
* Hierarchical display
* Real-time updates - when something changes in ETCD, the web UI is automatically updated
* Editing ETCD contents (if enabled)

## Running

The application is designed to be run in Docker.

For example, the following starts `etcd` and `etcdv3-browser` in Docker:

```
docker network create my_net
docker run -d --name etcd -p 2379:2379 --net my_net quay.io/coreos/etcd:v3.6.7 /usr/local/bin/etcd --listen-client-urls http://0.0.0.0:2379 --advertise-client-urls=http://127.0.0.1:2379
docker run -d --name etcdv3-browser --net my_net -p 8081:8081 -e HTTP_PORT=8081 -e ETCD=etcd:2379 -e EDITABLE=1 rustyx/etcdv3-browser
```

Open http://localhost:8081

If port 8081 is occupied, change all instances of 8081 above to some other port.

### Configuration

Environment variables:

| variable    | description                             | default                                       |
| ----------- | --------------------------------------- | --------------------------------------------- |
| `HTTP_PORT` | listen port                             | `8081`                                        |
| `ETCD`      | etcd endpoint                           | `etcd:2379`                                   |
| `CORS`      | allowed origins                         | `http://localhost:*`                          |
| `EDITABLE`  | set to `1` to enable edit functionality | `0`                                           |
| `PREFIX`    | only browse keys under a given prefix   | ``                                            |
| `USERNAME`  | optionally send a username to etcd      | `<empty>`                                     |
| `PASSWORD`  | optionally send a password to etcd      | `<empty>`                                     |

## Development environment

Initial setup: install Go 1.24+, Node.js 22+.

### Backend

```
cd backend
go build
./etcdv3-browser
```

### Frontend

```
cd frontend
npm run serve
```

### Running unit tests

```
cd backend
go test ./...
```

```
cd frontend
npm run test:unit
```

### Lints and code quality checks

```
cd frontend
npm run lint
```

### Building a Docker image

```
docker build . -t rustyx/etcdv3-browser
```
