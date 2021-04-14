package upload

import (
    "context"
    "net"
    "fmt"
    "os"
    "io"

    "google.golang.org/grpc"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"

    pb "github.com/oren12321/gogrpcft/internal/proto"
)

// Server represents a server side for files uploading.
type Server struct {
    pb.UnimplementedFilesTransferServer

    sentErr chan error
    ctx context.Context
    address string

    grpcser *grpc.Server
}

// NewServer returns new Server.
func NewServer(ctx context.Context, address string) *Server {
    return &Server{ctx: ctx, address: address, sentErr: make(chan error)}
}

func getData(packet *pb.Packet) ([]byte, error) {
    switch x := packet.PacketOptions.(type) {
    case *pb.Packet_FileInfo:
        return nil, fmt.Errorf("not a file info packat")
    case *pb.Packet_Chunk:
        return x.Chunk.Data, nil
    default:
        return nil, fmt.Errorf("unknown packat option")
    }
}

func getPath(packet *pb.Packet) (string, error) {
    switch x := packet.PacketOptions.(type) {
    case *pb.Packet_FileInfo:
        return x.FileInfo.Path, nil
    case *pb.Packet_Chunk:
        return "", fmt.Errorf("not a chunk packet")
    default:
        return "", fmt.Errorf("unknown packat option")
    }
}

func (s *Server) Upload(stream pb.FilesTransfer_UploadServer) error {

    packet, err := stream.Recv()
    if err != nil {
        errMsg := fmt.Sprintf("failed to receive first packet")
        s.sentErr <- fmt.Errorf(errMsg)
        return status.Errorf(codes.Internal, errMsg)
    }

    dst, err := getPath(packet)
    if err != nil {
        stream.SendAndClose(&pb.Status{
            Success: false,
            Msg: "first packet is not file info",
        })
        s.sentErr <- nil
        return nil
    }

    f, err := os.Create(dst)
    if err != nil {
        stream.SendAndClose(&pb.Status{
            Success: false,
            Msg: fmt.Sprintf("failed to create file %s: %v", dst, err),
        })
        s.sentErr <- nil
        return nil
    }
    defer f.Close()

    for {
        packet, err := stream.Recv()
        if err == io.EOF {
            break
        }
        if err != nil {
            errMsg := fmt.Sprintf("gRPC failed to receive: %v", err)
            s.sentErr <- fmt.Errorf(errMsg)
            return status.Errorf(codes.Internal, errMsg)
        }

        data, err := getData(packet)
        if err != nil {
            stream.SendAndClose(&pb.Status{
                Success: false,
                Msg: "received packet is not chuck",
            })
            s.sentErr <- nil
            return nil
        }
        size := len(data)

        if _, err := f.Write(data[:size]); err != nil {
            errMsg := fmt.Sprintf("failed to write chunk: %v", err)
            s.sentErr <- fmt.Errorf(errMsg)
            return status.Errorf(codes.Internal, errMsg)
        }
    }

    stream.SendAndClose(&pb.Status{
        Success: true,
        Msg: fmt.Sprintf("file upload succeeded: %s", dst),
    })
    s.sentErr <- nil
    return nil
}

// Start starts the server.
func (s *Server) Start(outerrch chan<- error) {
    lis, err := net.Listen("tcp", s.address)
    if err != nil {
        outerrch <- fmt.Errorf("Failed to listen: %v", err)
    }

    s.grpcser = grpc.NewServer()
    pb.RegisterFilesTransferServer(s.grpcser, s)

    go func() {
        s.grpcser.Serve(lis)
    }()

    for {
        select {
        case err := <-s.sentErr:
            outerrch <- err
        case <-s.ctx.Done():
            outerrch <- s.ctx.Err()
            return
        }
    }
}

// Stop stops the server.
func (s* Server) Stop() {
    s.grpcser.Stop()
}

