package upload

import (
    "context"
    "fmt"
    "io"
    "os"

    "google.golang.org/grpc"

    pb "github.com/oren12321/gogrpcft/internal/proto"
)

const (
    chunkSize = 2048
)

// RunClient uploads a file from src to remote via address.
func RunClient(ctx context.Context, address, src, remote string) error {

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

    conn, err := grpc.Dial(address, grpc.WithInsecure())
    if err != nil {
        return fmt.Errorf("gRPC connect failed: %v", err)
    }
    defer conn.Close()

    client := pb.NewFilesTransferClient(conn)
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

