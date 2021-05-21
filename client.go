// Package gogrpcft provides bytes transferring services via gRPC.
package gogrpcft

import (
    "context"
    "fmt"
    "io"

    "google.golang.org/grpc"

    "google.golang.org/protobuf/proto"
    "google.golang.org/protobuf/types/known/anypb"

    pb "github.com/oren12321/gogrpcft/v3/internal/proto"
)

// CreateTransferClient returns gRPC client given a connection.
func CreateTransferClient(conn *grpc.ClientConn) pb.TransferClient {
    return pb.NewTransferClient(conn)
}

// Receive downloads bytes from destination to source.
func Receive(client pb.TransferClient, ctx context.Context, streamerMsg, receiverMsg proto.Message, receiver BytesReceiver) (errout error) {

    any, err := anypb.New(streamerMsg)
    if err != nil {
        return fmt.Errorf("failed to create 'Any' from streamer message: %v", err)
    }

    req := &pb.Info{
        Msg: any,
    }

    stream, err := client.Receive(ctx, req)
    if err != nil {
        return fmt.Errorf("failed to fetch stream: %v", err)
    }

    if receiver == nil {
        return fmt.Errorf("receiver is nil")
    }

    if err := receiver.Init(receiverMsg); err != nil {
        return fmt.Errorf("failed to init receiver: %v", err)
    }

    defer func() {
        if err := receiver.Finalize(); err != nil {
            if errout == nil {
                errout = fmt.Errorf("failed to finalize receiver: %v", err)
            }
        }
    }()

    errch := make(chan error)

    go func() {
        for {
            res, err := stream.Recv()
            if err == io.EOF {
                errch <- nil
                return
            }
            if err != nil {
                errch <- fmt.Errorf("failed to receive: %v", err)
                return
            }

            data := res.Chunk.Data
            size := len(res.Chunk.Data)

            metadata, err := res.Info.Msg.UnmarshalNew()
            if err != nil {
                errch <- fmt.Errorf("failed to unmarshal 'Info.Msg': %v", err)
                return
            }

            if err := receiver.Push(data[:size], metadata); err != nil {
                errch <- fmt.Errorf("failed to push data to receiver: %v", err)
                return
            }
        }
    }()

    select {
    case err := <-errch:
        return err
    case <-ctx.Done():
        return ctx.Err()
    }
}

// Send uploads bytes from srouce to destination.
func Send(client pb.TransferClient, ctx context.Context, streamerMsg, receiverMsg proto.Message, streamer BytesStreamer) (errout error) {

    stream, err := client.Send(ctx)
    if err != nil {
        return fmt.Errorf("failed to fetch stream: %v", err)
    }

    if streamer == nil {
        return fmt.Errorf("streamer is nil")
    }

    if err := streamer.Init(streamerMsg); err != nil {
        return fmt.Errorf("failed to init streamer: %v", err)
    }

    defer func() {
        if err := streamer.Finalize(); err != nil {
            if errout == nil {
                errout = fmt.Errorf("failed to finalize stream: %v", err)
            }
        }
    }()

    any, err := anypb.New(receiverMsg)
    if err != nil {
        return fmt.Errorf("failed to create 'Any' from receiver message: %v", err)
    }

    req := &pb.Packet{
        Info: &pb.Info{
            Msg: any,
        },
        Chunk: &pb.Chunk{
            Data: nil,
        },
    }

    if err := stream.Send(req); err != nil {
        return fmt.Errorf("failed to send packet with 'Info': %v", err)
    }

    errch := make(chan error)

    go func() {

        for streamer.HasNext() {
            buf, metadata, err := streamer.GetNext()
            if err != nil {
                errch <- fmt.Errorf("failed to read from streamer: %v", err)
                return
            }

            if metadata == nil {
                errch <- fmt.Errorf("metadata can not be nil")
                return
            }

            any, err := anypb.New(metadata)
            if err != nil {
                errch <- fmt.Errorf("failed to create 'Any' from receiver metadata message: %v", err)
                return
            }

            req := &pb.Packet{
                Info: &pb.Info{
                    Msg: any,
                },
                Chunk: &pb.Chunk{
                    Data: buf,
                },
            }

            if err := stream.Send(req); err != nil {
                errch <- fmt.Errorf("failed to send 'Chunk' packet: %v", err)
                return
            }
        }

        _, err := stream.CloseAndRecv()
        if err != nil {
            errch <- fmt.Errorf("failed to close and recevie: %v", err)
        } else {
            errch <- nil
        }
    }()

    select {
    case err := <-errch:
        return err
    case <-ctx.Done():
        return ctx.Err()
    }
}
