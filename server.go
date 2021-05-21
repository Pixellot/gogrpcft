package gogrpcft

import (
    "io"
    "context"

    "google.golang.org/grpc"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"

    "google.golang.org/protobuf/types/known/anypb"
    "google.golang.org/protobuf/types/known/emptypb"

    pb "github.com/oren12321/gogrpcft/v3/internal/proto"
)

// SetBytesStreamer registers a bytes streamer to the gRPC server.
func (s *BytesTransferServer) SetBytesStreamer(streamer BytesStreamer) {
    s.streamer = streamer
}

// SetBytesReceiver registers a bytes receiver to the gRPC server.
func (s *BytesTransferServer) SetBytesReceiver(receiver BytesReceiver) {
    s.receiver = receiver
}

// Register registers a bytes transferring service to a given gRPC server.
func (s *BytesTransferServer) Register(server *grpc.Server) {
    pb.RegisterTransferServer(server, s)
}

// BytesTransferServer represents an implementation of bytes transferring gRPC server.
type BytesTransferServer struct {
    pb.UnimplementedTransferServer

    streamer BytesStreamer
    receiver BytesReceiver
}

// Receive is the bytes download implementation of the bytes transferring service.
// comment: should not be used directly.
func (s *BytesTransferServer) Receive(in *pb.Info, stream pb.Transfer_ReceiveServer) (errout error) {

    if stream.Context().Err() == context.Canceled {
        return status.Errorf(codes.Canceled, "client cancelled, abandoning")
    }

    if s.streamer == nil {
        return status.Errorf(codes.FailedPrecondition, "streamer is nil")
    }

    streamerMsg, err := in.Msg.UnmarshalNew()
    if err != nil {
        return status.Errorf(codes.InvalidArgument, "failed to unmarshal 'Info.Msg': %v", err)
    }

    if err := s.streamer.Init(streamerMsg); err != nil {
        return status.Errorf(codes.FailedPrecondition, "failed to init streamer: %v", err)
    }
    defer func() {
        if err := s.streamer.Finalize(); err != nil {
            if errout == nil {
                errout = status.Errorf(codes.FailedPrecondition, "failed to finalize streamer: %v", err)
            }
        }
    }()

	for s.streamer.HasNext() {
		buf, metadata, err := s.streamer.GetNext()
        if err != nil {
            return status.Errorf(codes.FailedPrecondition, "failed to read chunk from streamer: %v", err)
        }

        any, err := anypb.New(metadata)
        if err != nil {
            return status.Errorf(codes.InvalidArgument, "failed to create 'Any' from streamer metadata message: %v", err)
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
            return status.Errorf(codes.Internal, "failed to send chunk: %v", err)
        }
	}

    return nil
}

// Send is the bytes upload implementation of the bytes transferring service.
// comment: should not be used directly.
func (s *BytesTransferServer) Send(stream pb.Transfer_SendServer) (errout error) {

    if stream.Context().Err() == context.Canceled {
        return status.Errorf(codes.Canceled, "client cancelled, abandoning")
    }

    if s.receiver == nil {
        return status.Errorf(codes.FailedPrecondition, "receiver is nil")
    }

    packet, err := stream.Recv()
    if err != nil {
        return status.Errorf(codes.Internal, "failed to receive first packet: %v", err)
    }

    receiverMsg, err := packet.Info.Msg.UnmarshalNew()
    if err != nil {
        return status.Errorf(codes.InvalidArgument, "failed to unmarshal 'Info.Msg': %v", err)
    }

    if err := s.receiver.Init(receiverMsg); err != nil {
        return status.Errorf(codes.FailedPrecondition, "failed to init receiver: %v", err)
    }
    defer func() {
        if err := s.receiver.Finalize(); err != nil {
            if errout == nil {
                errout = status.Errorf(codes.FailedPrecondition, "failed to finalize receiver: %v", err)
            }
        }

        stream.SendAndClose(&emptypb.Empty{})
    }()

    for {
        packet, err := stream.Recv()
        if err == io.EOF {
            break
        }
        if err != nil {
            return status.Errorf(codes.Internal, "failed to receive packet: %v", err)
        }

        data := packet.Chunk.Data
        size := len(data)

        metadata, err := packet.Info.Msg.UnmarshalNew()
        if err != nil {
            return status.Errorf(codes.InvalidArgument, "failed to unmarshal 'Info.Msg': %v", err)
        }

        if err := s.receiver.Push(data[:size], metadata); err != nil {
            return status.Errorf(codes.FailedPrecondition, "failed to push chunk to receiver: %v", err)
        }
    }

    return nil
}

