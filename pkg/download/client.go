package download

import (
    "context"
    "fmt"
    "io"
    "os"

    "google.golang.org/grpc"

    pb "github.com/oren12321/gogrpcft/internal/proto"
)

func RunClient(ctx context.Context, address, remote, dst string) error {

    conn, err := grpc.Dial(address, grpc.WithInsecure())
    if err != nil {
        return fmt.Errorf("gRPC connect failed: %v", err)
    }
    defer conn.Close()

    client := pb.NewFilesTransferClient(conn)
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

