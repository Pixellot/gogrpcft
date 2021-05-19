package gogrpcft

import (
    "log"
    "context"
    "net"

    "google.golang.org/grpc"
    "google.golang.org/grpc/test/bufconn"
)

var lis *bufconn.Listener
var bts *BytesTransferServer

func init() {
    lis = bufconn.Listen(1024 * 1024)
    s := grpc.NewServer()
    bts = &BytesTransferServer{}
    bts.Register(s)
    go func() {
        if err := s.Serve(lis); err != nil {
            log.Fatalf("failed to listen: %v", err)
        }
    }()
}

func dialer(context.Context, string) (net.Conn, error) {
    return lis.Dial()
}

