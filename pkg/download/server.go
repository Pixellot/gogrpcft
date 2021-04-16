package download

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

const (
    chunkSize = 2048
)

// Server represents a server side for files downloading.
type Server struct {
    pb.UnimplementedFilesTransferServer

    ctx context.Context
    address string

    grpcser *grpc.Server
}

// NewServer returns new Server.
func NewServer(ctx context.Context, address string) *Server {
    return &Server{ctx: ctx, address: address}
}

func (s *Server) Download(in *pb.FileInfo, stream pb.FilesTransfer_DownloadServer) error {

    info, err := os.Stat(in.Path)
    if os.IsNotExist(err) {
        errMsg := fmt.Sprintf("path not found: %s", in.Path)
        return status.Errorf(codes.FailedPrecondition, errMsg)
    }
    if info.IsDir() {
        errMsg := fmt.Sprintf("unable to download directory: %s", in.Path)
        return status.Errorf(codes.FailedPrecondition, errMsg)
    }
    if info.Size() == 0 {
        errMsg := fmt.Sprintf("file is empty: %s", in.Path)
        return status.Errorf(codes.FailedPrecondition, errMsg)
    }

	f, err := os.Open(in.Path)
	if err != nil {
		errMsg := fmt.Sprintf("failed to open file %s: %v", in.Path, err)
        return status.Errorf(codes.FailedPrecondition, errMsg)
	}
	defer f.Close()

	buf := make([]byte, chunkSize)

	for {
		n, err := f.Read(buf)
        if err == io.EOF {
            break
        }
        if err != nil {
            errMsg := fmt.Sprintf("failed to read chunk: %v", err)
            return status.Errorf(codes.Internal, errMsg)
        }

        buf = buf[:n]
        if err := stream.Send(&pb.Chunk{Data: buf}); err != nil {
            errMsg := fmt.Sprintf("failed to send chunk: %v", err)
            return status.Errorf(codes.Internal, errMsg)
        }
	}

    return nil
}

// Start starts the server.
func (s *Server) Start() error {
    lis, err := net.Listen("tcp", s.address)
    if err != nil {
        return fmt.Errorf("Failed to listen: %v", err)
    }

    s.grpcser = grpc.NewServer()
    pb.RegisterFilesTransferServer(s.grpcser, s)

    go func() {
        s.grpcser.Serve(lis)
    }()

    select {
    case <-s.ctx.Done():
        return s.ctx.Err()
    }
}

// Stop stops the server.
func (s* Server) Stop() {
    s.grpcser.Stop()
}

