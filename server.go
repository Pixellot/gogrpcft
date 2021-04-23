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
func (s *BytesTransferServer) Receive(in *pb.Info, stream pb.Transfer_ReceiveServer) error {

    if stream.Context().Err() == context.Canceled {
        return status.Errorf(codes.Canceled, "client cancelled, abandoning")
    }

    if s.streamer == nil {
        return status.Errorf(codes.FailedPrecondition, "streamer is nil")
    }

    if err := s.streamer.Init(in.Msg); err != nil {
        return status.Errorf(codes.FailedPrecondition, "streamer init failed")
    }

	for s.streamer.HasNext() {
		buf, err := s.streamer.GetNext()
        if err != nil {
            errMsg := fmt.Sprintf("failed to read chunk: %v", err)
            return status.Errorf(codes.Internal, errMsg)
        }

        if err := stream.Send(&pb.Chunk{Data: buf}); err != nil {
            errMsg := fmt.Sprintf("failed to send chunk: %v", err)
            return status.Errorf(codes.Internal, errMsg)
        }
	}

    if err := s.streamer.Finalize(); err != nil {
        return status.Errorf(codes.FailedPrecondition, "streamer finalize failed")
    }

    return nil
}

// Send is the file upload implementation of the bytes transferring service.
// comment: should not be used directly.
func (s *BytesTransferServer) Send(stream pb.Transfer_SendServer) error {

    if stream.Context().Err() == context.Canceled {
        return status.Errorf(codes.Canceled, "client cancelled, abandoning")
    }

    if s.receiver == nil {
        return status.Errorf(codes.FailedPrecondition, "receiver is nil")
    }

    packet, err := stream.Recv()
    if err != nil {
        errMsg := fmt.Sprintf("failed to receive first packet")
        return status.Errorf(codes.Internal, errMsg)
    }

    dst, err := getPath(packet)
    if err != nil {
        stream.SendAndClose(&pb.Status{
            Success: false,
            Desc: "first packet is not file info",
        })
        return nil
    }

    if err := s.receiver.Init(dst); err != nil {
        return status.Errorf(codes.FailedPrecondition, "receiver init failed")
    }

    for {
        packet, err := stream.Recv()
        if err == io.EOF {
            break
        }
        if err != nil {
            errMsg := fmt.Sprintf("gRPC failed to receive: %v", err)
            return status.Errorf(codes.Internal, errMsg)
        }

        data, err := getData(packet)
        if err != nil {
            stream.SendAndClose(&pb.Status{
                Success: false,
                Desc: "received packet is not chuck",
            })
            return nil
        }
        size := len(data)

        if err := s.receiver.Push(data[:size]); err != nil {
            errMsg := fmt.Sprintf("failed to write chunk: %v", err)
            return status.Errorf(codes.Internal, errMsg)
        }
    }

    if err := s.receiver.Finalize(); err != nil {
        return status.Errorf(codes.FailedPrecondition, "receiver finalize failed")
    }

    stream.SendAndClose(&pb.Status{
        Success: true,
        Desc: fmt.Sprintf("file upload succeeded: %s", dst),
    })
    return nil
}


func getData(packet *pb.Packet) ([]byte, error) {
    switch x := packet.PacketOptions.(type) {
    case *pb.Packet_Info:
        return nil, fmt.Errorf("not a info packat")
    case *pb.Packet_Chunk:
        return x.Chunk.Data, nil
    default:
        return nil, fmt.Errorf("unknown packat option")
    }
}

func getPath(packet *pb.Packet) (string, error) {
    switch x := packet.PacketOptions.(type) {
    case *pb.Packet_Info:
        return x.Info.Msg, nil
    case *pb.Packet_Chunk:
        return "", fmt.Errorf("not a chunk packet")
    default:
        return "", fmt.Errorf("unknown packat option")
    }
}
