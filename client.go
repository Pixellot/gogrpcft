// Package gogrpcft provides files transferring services via gRPC.
package gogrpcft

import (
    "context"
    "fmt"
    "io"

    "google.golang.org/grpc"

    "google.golang.org/protobuf/proto"
    "google.golang.org/protobuf/types/known/anypb"

    pb "github.com/oren12321/gogrpcft/v2/internal/proto"
)

// CreateFilesTransferClient returns gRPC client given a connection.
func CreateTransferClient(conn *grpc.ClientConn) pb.TransferClient {
    return pb.NewTransferClient(conn)
}

// DownloadBytes downloads bytes from destination to source.
func DownloadBytes(client pb.TransferClient, ctx context.Context, streamerMsg, receiverMsg proto.Message, receiver BytesReceiver) error {

    any, err := anypb.New(streamerMsg)
    if err != nil {
        return fmt.Errorf("failed to create any: %v", err)
    }

    req := &pb.Info{
        Msg: any,
    }

    stream, err := client.Receive(ctx, req)
    if err != nil {
        return fmt.Errorf("gRPC stream fetch failed: %v", err)
    }

    if receiver == nil {
        return fmt.Errorf("receiver is nil")
    }

    if err := receiver.Init(receiverMsg); err != nil {
        return fmt.Errorf("receiver init failed: %v", err)
    }

    errch := make(chan error)

    go func() {
        for {
            res, err := stream.Recv()
            if err == io.EOF {
                if err := receiver.Finalize(); err != nil {
                    errch <- fmt.Errorf("receiver finalize failed: %v", err)
                } else {
                    errch <- nil
                }
                return
            }
            if err != nil {
                errch <- fmt.Errorf("gRPC failed to receive: %v", err)
                return
            }

            data := res.Data
            size := len(res.Data)
            if err := receiver.Push(data[:size]); err != nil {
                errch <- fmt.Errorf("failed to write chunk: %v", err)
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

// UploadBytes uploads bytes from srouce to destination.
func UploadBytes(client pb.TransferClient, ctx context.Context, streamerMsg, receiverMsg proto.Message, streamer BytesStreamer) error {

    stream, err := client.Send(ctx)
    if err != nil {
        return fmt.Errorf("gRPC stream fetch failed: %v", err)
    }

    if streamer == nil {
        return fmt.Errorf("streamer is nil")
    }

    if err := streamer.Init(streamerMsg); err != nil {
        return fmt.Errorf("streamer init failed")
    }

    any, err := anypb.New(receiverMsg)
    if err != nil {
        return fmt.Errorf("failed to create any: %v", err)
    }

    req := &pb.Packet{
        PacketOptions: &pb.Packet_Info{
            Info: &pb.Info{
                Msg: any,
            },
        },
    }

    if err := stream.Send(req); err != nil {
        return fmt.Errorf("failed to send packet path info: %v", err)
    }

    errch := make(chan error)

    go func() {

        for streamer.HasNext() {
            buf, err := streamer.GetNext()
            if err != nil {
                errch <- fmt.Errorf("failed to read chunk: %v", err)
                return
            }

            req := &pb.Packet{
                PacketOptions: &pb.Packet_Chunk{
                    Chunk: &pb.Chunk{
                        Data: buf,
                    },
                },
            }

            if err := stream.Send(req); err != nil {
                errch <- fmt.Errorf("failed to send chunk: %v", err)
                return
            }
        }

        res, err := stream.CloseAndRecv()
        if err != nil {
            errch <- fmt.Errorf("failed to close and receive status: %v", err)
        } else if !res.Success {
            errch <- fmt.Errorf("bad response from server: %s", res.Desc)
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
