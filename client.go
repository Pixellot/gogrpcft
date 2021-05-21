// Package gogrpcft provides files transferring services via gRPC.
package gogrpcft

import (
    "context"
    "fmt"
    "io"
    "os"

    "google.golang.org/grpc"

    pb "github.com/oren12321/gogrpcft/v2/internal/proto"
)

// CreateFilesTransferClient returns gRPC client given a connection.
func CreateFilesTransferClient(conn *grpc.ClientConn) pb.FilesTransferClient {
    return pb.NewFilesTransferClient(conn)
}

// DownloadFile downloads a file from destination to source.
func DownloadFile(client pb.FilesTransferClient, ctx context.Context, from, to string) error {

    req := &pb.FileInfo{Path: from}
    stream, err := client.Download(ctx, req)
    if err != nil {
        return fmt.Errorf("gRPC stream fetch failed: %v", err)
    }

    f, err := os.Create(to)
    if err != nil {
        return fmt.Errorf("failed to create downloaded file '%s': %v", to, err)
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

// UploadFile uploads file from srouce to destination.
func UploadFile(client pb.FilesTransferClient, ctx context.Context, from, to string) error {

    info, err := os.Stat(from)
    if os.IsNotExist(err) {
        return fmt.Errorf("path not found: %s", from)
    }
    if info.IsDir() {
        return fmt.Errorf("unable to upload directory, '%s' must be file", from)
    }

    stream, err := client.Upload(ctx)
    if err != nil {
        return fmt.Errorf("gRPC stream fetch failed: %v", err)
    }

    req := &pb.Packet{
        PacketOptions: &pb.Packet_FileInfo{
            FileInfo: &pb.FileInfo{
                Path: to,
            },
        },
    }
    if err := stream.Send(req); err != nil {
        return fmt.Errorf("failed to send packet path info: %v", err)
    }

	f, err := os.Open(from)
	if err != nil {
		return fmt.Errorf("failed to open file '%s': %v", from, err)
	}
	defer f.Close()

    errch := make(chan error)

    go func() {

        buf := make([]byte, 2048)

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

        // gRPC reponse ignored, and should be removed in next release.
        //res, err := stream.CloseAndRecv()
        _, err := stream.CloseAndRecv()
        if err != nil {
            //errch <- fmt.Errorf("failed to close and receive with success=%v and msg='%s': %v", res.Success, res.Msg, err)
            errch <- fmt.Errorf("failed to close and receive: %v", err)
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

