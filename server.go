package gogrpcft

import (
    "fmt"
    "os"
    "io"

    "google.golang.org/grpc"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"

    pb "github.com/oren12321/gogrpcft/internal/proto"
)

func RegisterFilesTransferServer(server *grpc.Server) {
    pb.RegisterFilesTransferServer(server, &filesTransferServer{})
}

type filesTransferServer struct {
    pb.UnimplementedFilesTransferServer
}

func (s *filesTransferServer) Download(in *pb.FileInfo, stream pb.FilesTransfer_DownloadServer) error {

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

    chunkSize := 2048
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

func (s *filesTransferServer) Upload(stream pb.FilesTransfer_UploadServer) error {

    packet, err := stream.Recv()
    if err != nil {
        errMsg := fmt.Sprintf("failed to receive first packet")
        return status.Errorf(codes.Internal, errMsg)
    }

    dst, err := getPath(packet)
    if err != nil {
        stream.SendAndClose(&pb.Status{
            Success: false,
            Msg: "first packet is not file info",
        })
        return nil
    }

    f, err := os.Create(dst)
    if err != nil {
        stream.SendAndClose(&pb.Status{
            Success: false,
            Msg: fmt.Sprintf("failed to create file %s: %v", dst, err),
        })
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
            return status.Errorf(codes.Internal, errMsg)
        }

        data, err := getData(packet)
        if err != nil {
            stream.SendAndClose(&pb.Status{
                Success: false,
                Msg: "received packet is not chuck",
            })
            return nil
        }
        size := len(data)

        if _, err := f.Write(data[:size]); err != nil {
            errMsg := fmt.Sprintf("failed to write chunk: %v", err)
            return status.Errorf(codes.Internal, errMsg)
        }
    }

    stream.SendAndClose(&pb.Status{
        Success: true,
        Msg: fmt.Sprintf("file upload succeeded: %s", dst),
    })
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

