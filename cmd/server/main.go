package main

import (
    "flag"
    "net"
    "log"

    "google.golang.org/grpc"

    ft "github.com/oren12321/gogrpcft/v2"
    "github.com/oren12321/gogrpcft/v2/receiver"
    "github.com/oren12321/gogrpcft/v2/streamer"
)

func main() {

    address := flag.String("address", "127.0.0.1:8080", "server address")

    flag.Parse()

    lis, err := net.Listen("tcp", *address)
    if err != nil {
        log.Fatalf("failed to listen: %v", err)
    }

    s := grpc.NewServer()
    service := &ft.BytesTransferServer{}
    service.Register(s)
    service.SetBytesReceiver(&receiver.FileReceiver{})
    service.SetBytesStreamer(&streamer.FileStreamer{})

    if err := s.Serve(lis); err != nil {
        log.Fatalf("failed to serve: %v", err)
    }
}

