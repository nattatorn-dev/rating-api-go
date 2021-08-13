# Install

```bash
$ go mod init github.com/nattatorn-dev/rating-api
$ go mod tidy
```

## Run

```bash
$ go run *.go
```

## Live Reload

```bash
$ air
```

## Docker

```bash
$ docker build -t nattatorn-dev/rating-api .
$ docker run --rm -p 8080:80 nattatorn-dev/rating-api
```
