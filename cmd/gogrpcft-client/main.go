package main

import (
    "flag"
    "log"
    "time"
    "os"
    "context"

    "google.golang.org/grpc"

    ft "github.com/oren12321/gogrpcft/v2"

    pb "github.com/oren12321/gogrpcft/v2/interface/file/proto"
    fi "github.com/oren12321/gogrpcft/v2/interface/file"
)

func main() {

    downloadCmd := flag.NewFlagSet("download", flag.ExitOnError)
    downloadAddress := downloadCmd.String("address", "127.0.0.1:8080", "server address")
    downloadFrom := downloadCmd.String("from", "", "remote file path")
    downloadTo := downloadCmd.String("to", "", "download destination path")

    uploadCmd := flag.NewFlagSet("upload", flag.ExitOnError)
    uploadAddress := uploadCmd.String("address", "127.0.0.1:8080", "server address")
    uploadFrom := uploadCmd.String("from", "", "file path to upload")
    uploadTo := uploadCmd.String("to", "", "remote destination file path")

    if len(os.Args) < 2 {
        log.Fatal("expected 'download' or 'upload' command")
    }

    flag.Parse()

    switch os.Args[1] {

    case "download":
        downloadCmd.Parse(os.Args[2:])
        conn := dial(*downloadAddress)
        defer conn.Close()
        c := ft.CreateTransferClient(conn)
        fromInfo := &pb.File{Path: *downloadFrom}
        toInfo := &pb.File{Path: *downloadTo}
        if err := ft.Receive(c, context.Background(), fromInfo, toInfo, &fi.FileReceiver{}); err != nil {
            log.Fatalf("client failed: %v", err)
        }
    case "upload":
        uploadCmd.Parse(os.Args[2:])
        conn := dial(*uploadAddress)
        defer conn.Close()
        c := ft.CreateTransferClient(conn)
        fromInfo := &pb.File{Path: *uploadFrom}
        toInfo := &pb.File{Path: *uploadTo}
        if err := ft.Send(c, context.Background(), fromInfo, toInfo, &fi.FileStreamer{}); err != nil {
            log.Fatalf("client failed: %v", err)
        }
    default:
        log.Fatal("expected 'download' or 'upload' command")
    }

}

func dial(address string) *grpc.ClientConn {
    conn, err := grpc.Dial(address, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithTimeout(time.Minute))
    if err != nil {
        log.Fatalf("did not connect: %v", err)
    }
    return conn
}

