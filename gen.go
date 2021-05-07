//go:generate protoc -I internal/proto --go_out=internal/proto --go_opt=paths=source_relative --go-grpc_out=internal/proto --go-grpc_opt=paths=source_relative internal/proto/io.proto

//go:generate protoc -I interface/file/proto --go_out=interface/file/proto --go_opt=paths=source_relative --go-grpc_out=interface/file/proto --go-grpc_opt=paths=source_relative interface/file/proto/file.proto

package gogrpcft

