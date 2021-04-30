package gogrpcft

import (
    "fmt"
    "io"
    "context"

    "google.golang.org/grpc"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"

    pb "github.com/oren12321/gogrpcft/v2/internal/proto"
)

func (s *BytesTransferServer) SetBytesStreamer(streamer BytesStreamer) {
    s.streamer = streamer
}

func (s *BytesTransferServer) SetBytesReceiver(receiver BytesReceiver) {
    s.receiver = receiver
}

// Register registers a bytes transferring service to a given gRPC server.
func (s *BytesTransferServer) Register(server *grpc.Server) {
    pb.RegisterTransferServer(server, s)
}

type BytesTransferServer struct {
    pb.UnimplementedTransferServer

    streamer BytesStreamer
    receiver BytesReceiver
}

// Receive is the file download implementation of the bytes transferring service.
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
        return status.Errorf(codes.FailedPrecondition, "failed to init streamer")
    }
    defer func() {
        if err := s.streamer.Finalize(); err != nil {
            if errout == nil {
                errout = status.Errorf(codes.FailedPrecondition, "failed to finalize streamer")
            }
        }
    }()

	for s.streamer.HasNext() {
		buf, err := s.streamer.GetNext()
        if err != nil {
            return status.Errorf(codes.FailedPrecondition, "failed to read chunk from streamer: %v", err)
        }

        if err := stream.Send(&pb.Chunk{Data: buf}); err != nil {
            return status.Errorf(codes.Internal, "failed to send chunk: %v", err)
        }
	}

    return nil
}

// Send is the file upload implementation of the bytes transferring service.
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

    info, err := getInfo(packet)
    if err != nil {
        stream.SendAndClose(&pb.Status{
            Success: false,
            Desc: "first packet is not 'Info'",
        })
        return nil
    }

    receiverMsg, err := info.Msg.UnmarshalNew()
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
        } else {
            stream.SendAndClose(&pb.Status{
                Success: true,
                Desc: "send finished successfuly",
            })
        }
    }()

    for {
        packet, err := stream.Recv()
        if err == io.EOF {
            break
        }
        if err != nil {
            return status.Errorf(codes.Internal, "failed to receive packet: %v", err)
        }

        chunk, err := getChunk(packet)
        if err != nil {
            stream.SendAndClose(&pb.Status{
                Success: false,
                Desc: "packet is not 'Chunk'",
            })
            return nil
        }
        data := chunk.Data
        size := len(data)

        if err := s.receiver.Push(data[:size]); err != nil {
            return status.Errorf(codes.FailedPrecondition, "failed to push chunk to receiver: %v", err)
        }
    }

    return nil
}


func getChunk(packet *pb.Packet) (*pb.Chunk, error) {
    switch x := packet.PacketOptions.(type) {
    case *pb.Packet_Info:
        return nil, fmt.Errorf("not an 'Info' packat")
    case *pb.Packet_Chunk:
        return x.Chunk, nil
    default:
        return nil, fmt.Errorf("unknown packat option")
    }
}

func getInfo(packet *pb.Packet) (*pb.Info, error) {
    switch x := packet.PacketOptions.(type) {
    case *pb.Packet_Info:
        return x.Info, nil
    case *pb.Packet_Chunk:
        return nil, fmt.Errorf("not a 'Chunk' packet")
    default:
        return nil, fmt.Errorf("unknown packat option")
    }
}

