all:
	go test -v github.com/dolab/redis-go/metrics
	go test -v -race github.com/dolab/redis-go/metrics
	go test -v github.com/dolab/redis-go
	go test -v -race github.com/dolab/redis-go

gocover:
	go test -cover github.com/dolab/redis-go/metrics
	go test -cover github.com/dolab/redis-go
