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

type Server struct {
    pb.UnimplementedFilesTransferServer

    sentErr chan error
    ctx context.Context
    address string

    grpcser *grpc.Server
}

func NewServer(ctx context.Context, address string) *Server {
    return &Server{ctx: ctx, address: address, sentErr: make(chan error)}
}

func (s *Server) Download(in *pb.FileInfo, stream pb.FilesTransfer_DownloadServer) error {

    info, err := os.Stat(in.Path)
    if os.IsNotExist(err) {
        errMsg := fmt.Sprintf("path not found: %s", in.Path)
        s.sentErr <- fmt.Errorf(errMsg)
        return status.Errorf(codes.FailedPrecondition, errMsg)
    }
    if info.IsDir() {
        errMsg := fmt.Sprintf("unable to download directory: %s", in.Path)
        s.sentErr <- fmt.Errorf(errMsg)
        return status.Errorf(codes.FailedPrecondition, errMsg)
    }
    if info.Size() == 0 {
        errMsg := fmt.Sprintf("file is empty: %s", in.Path)
        s.sentErr <- fmt.Errorf(errMsg)
        return status.Errorf(codes.FailedPrecondition, errMsg)
    }

	f, err := os.Open(in.Path)
	if err != nil {
		errMsg := fmt.Sprintf("failed to open file %s: %v", in.Path, err)
        s.sentErr <- fmt.Errorf(errMsg)
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
            s.sentErr <- fmt.Errorf(errMsg)
            return status.Errorf(codes.Internal, errMsg)
        }

        buf = buf[:n]
        if err := stream.Send(&pb.Chunk{Data: buf}); err != nil {
            errMsg := fmt.Sprintf("failed to send chunk: %v", err)
            s.sentErr <- fmt.Errorf(errMsg)
            return status.Errorf(codes.Internal, errMsg)
        }
	}

    s.sentErr <- nil
    return nil
}

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
    case err := <-s.sentErr:
        return err
    case <-s.ctx.Done():
        return s.ctx.Err()
    }
}

func (s* Server) Stop() {
    s.grpcser.Stop()
}

