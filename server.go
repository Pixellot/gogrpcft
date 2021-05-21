package gogrpcft

import (
    "fmt"
    "os"
    "io"
    "context"

    "google.golang.org/grpc"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"

    pb "github.com/oren12321/gogrpcft/v2/internal/proto"
)

// RegisterFilesTransferServer registers a files transferring service to a given gRPC server.
func RegisterFilesTransferServer(server *grpc.Server) {
    pb.RegisterFilesTransferServer(server, &filesTransferServer{})
}

type filesTransferServer struct {
    pb.UnimplementedFilesTransferServer
}

// Download is the file download implementation of the files transferring service.
// comment: should not be used directly.
func (s *filesTransferServer) Download(in *pb.FileInfo, stream pb.FilesTransfer_DownloadServer) error {

    if stream.Context().Err() == context.Canceled {
        return status.Errorf(codes.Canceled, "client cancelled, abandoning")
    }

    info, err := os.Stat(in.Path)
    if os.IsNotExist(err) {
        return status.Errorf(codes.FailedPrecondition, "path '%s' not found", in.Path)
    }
    if info.IsDir() {
        return status.Errorf(codes.FailedPrecondition, "unable to download directory '%s'", in.Path)
    }

	f, err := os.Open(in.Path)
	if err != nil {
        return status.Errorf(codes.FailedPrecondition, "failed to open file '%s': %v", in.Path, err)
	}
	defer f.Close()

    chunkSize := 2048
	buf := make([]byte, chunkSize)

	for {
		n, err := f.Read(buf)
        if err == io.EOF {
            break
        }
        if err != nil {
            return status.Errorf(codes.FailedPrecondition, "failed to read chunk: %v", err)
        }

        buf = buf[:n]
        if err := stream.Send(&pb.Chunk{Data: buf}); err != nil {
            return status.Errorf(codes.Internal, "failed to send chunk: %v", err)
        }
	}

    return nil
}

// Upload is the file upload implementation of the files transferring service.
// comment: should not be used directly.
func (s *filesTransferServer) Upload(stream pb.FilesTransfer_UploadServer) error {

    if stream.Context().Err() == context.Canceled {
        return status.Errorf(codes.Canceled, "client cancelled, abandoning")
    }

    packet, err := stream.Recv()
    if err != nil {
        return status.Errorf(codes.Internal, "failed to receive first packet: %v", err)
    }

    dst, err := getPath(packet)
    if err != nil {
        //stream.SendAndClose(&pb.Status{
        //    Success: false,
        //    Msg: "first packet is not 'Info'",
        //})
        return status.Errorf(codes.InvalidArgument, "invalid first packet: %v", err)
    }

    f, err := os.Create(dst)
    if err != nil {
        //stream.SendAndClose(&pb.Status{
        //    Success: false,
        //    Msg: "file creation failed",
        //})
        return status.Errorf(codes.FailedPrecondition, "failed to create file '%s': %v", dst, err)
    }
    defer f.Close()

    for {
        packet, err := stream.Recv()
        if err == io.EOF {
            break
        }
        if err != nil {
            return status.Errorf(codes.Internal, "gRPC failed to receive: %v", err)
        }

        data, err := getData(packet)
        if err != nil {
            //stream.SendAndClose(&pb.Status{
            //    Success: false,
            //    Msg: "packet is not 'Chunk'",
            //})
            return status.Errorf(codes.InvalidArgument, "invalid packet: %v", err)
        }
        size := len(data)

        if _, err := f.Write(data[:size]); err != nil {
            return status.Errorf(codes.FailedPrecondition, "failed to write chunk: %v", err)
        }
    }

    //stream.SendAndClose(&pb.Status{
    //    Success: true,
    //    Msg: fmt.Sprintf("file '%s' upload finished successfuly", dst),
    //})
    stream.SendAndClose(&pb.Status{})
    return nil
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

