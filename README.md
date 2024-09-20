# etcdv3-browser

A simple etcd (v3) web-based browser.

![etcd browser](https://rustyx.org/temp/etcdv3-browser.png)

## Running

The application is designed to be run in Docker.

For example, the following starts `etcd` and `etcdv3-browser` in Docker:

```
docker network create my_net
docker run -d --name etcd -p 2379:2379 --net my_net quay.io/coreos/etcd
docker run -d --name etcdv3-browser -p 8081:8081 --net my_net -e ETCD=etcd:2379 -e EDITABLE=1 rustyx/etcdv3-browser
```

Open http://localhost:8081

### Configuration

Environment variables:

| variable    | description                             | default                                       |
| ----------- | --------------------------------------- | --------------------------------------------- |
| `HTTP_PORT` | listen port                             | `8081`                                        |
| `ETCD`      | etcd endpoint                           | `etcd:2379`                                   |
| `CORS`      | allowed origins                         | `http://localhost:8080,http://localhost:8081` |
| `EDITABLE`  | set to `1` to enable edit functionality | `0`                                           |
| `PREFIX`    | only browse keys under a given prefix   | ``                                            |
| `USERNAME`  | optionally send a username to etcd      | `<empty>`                                     |
| `PASSWORD`  | optionally send a password to etcd      | `<empty>`                                     |

## Development environment

Initial setup: install Go, Node.js, `npm install -g yarn`

### Backend

```
cd backend
go build
./etcdv3-browser
```

### Frontend

```
cd frontend
yarn serve
```

### Running unit tests

```
go test ./...
npm run test:unit
```

### Lints and code quality checks

```
npm run lint
```
