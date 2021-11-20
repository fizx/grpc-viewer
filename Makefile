build: proto
	go build ./...
	
proto:
	protoc --go_out=. --go_opt=module=github.com/fizx/grpc-viewer \
    --go-grpc_out=. --go-grpc_opt=module=github.com/fizx/grpc-viewer \
	example.proto

