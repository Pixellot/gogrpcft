package gogrpcft

import (
    "context"
    "fmt"
    "io"
    "os"

    "google.golang.org/grpc"

    pb "github.com/oren12321/gogrpcft/internal/proto"
)

func CreateFilesTransferClient(conn *grpc.ClientConn) pb.FilesTransferClient {
    return pb.NewFilesTransferClient(conn)
}

func DownloadFile(client pb.FilesTransferClient, ctx context.Context, remote, dst string) error {

    req := &pb.FileInfo{Path: remote}
    stream, err := client.Download(ctx, req)
    if err != nil {
        return fmt.Errorf("gRPC stream fetch failed: %v", err)
    }

    f, err := os.Create(dst)
    if err != nil {
        return fmt.Errorf("failed to create downloaded file: %v", err)
    }
    defer f.Close()

    errch := make(chan error)

    go func() {
        for {
            res, err := stream.Recv()
            if err == io.EOF {
                errch <- nil
                return
            }
            if err != nil {
                errch <- fmt.Errorf("gRPC failed to receive: %v", err)
                return
            }

            data := res.Data
            size := len(res.Data)
            if _, err := f.Write(data[:size]); err != nil {
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

func UploadFile(client pb.FilesTransferClient, ctx context.Context, src, remote string) error {

    info, err := os.Stat(src)
    if os.IsNotExist(err) {
        return fmt.Errorf("path not found: %s", src)
    }
    if info.IsDir() {
        return fmt.Errorf("unable to upload directory: %s", src)
    }
    if info.Size() == 0 {
        return fmt.Errorf("file is empty: %s", src)
    }

    stream, err := client.Upload(ctx)
    if err != nil {
        return fmt.Errorf("gRPC stream fetch failed: %v", err)
    }

    req := &pb.Packet{
        PacketOptions: &pb.Packet_FileInfo{
            FileInfo: &pb.FileInfo{
                Path: remote,
            },
        },
    }
    if err := stream.Send(req); err != nil {
        return fmt.Errorf("failed to send packet path info: %v", err)
    }

	f, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %v", src, err)
	}
	defer f.Close()

    errch := make(chan error)

    go func() {

        chunkSize := 2048
        buf := make([]byte, chunkSize)

        for {
            n, err := f.Read(buf)
            if err == io.EOF {
                break
            }
            if err != nil {
                errch <- fmt.Errorf("failed to read chunk: %v", err)
                return
            }

            buf = buf[:n]

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
            errch <- fmt.Errorf("bad response from server: %s", res.Msg)
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
