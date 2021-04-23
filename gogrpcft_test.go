package gogrpcft

import (
    "testing"
    "log"
    "context"
    "net"
    "os"
    "io/ioutil"
    "bytes"
    "path/filepath"

    "google.golang.org/grpc"
    "google.golang.org/grpc/test/bufconn"

    "github.com/oren12321/gogrpcft/v2/streamer"
    "github.com/oren12321/gogrpcft/v2/receiver"
)

var lis *bufconn.Listener
var bts *BytesTransferServer

func init() {
    lis = bufconn.Listen(1024 * 1024)
    s := grpc.NewServer()
    bts = &BytesTransferServer{}
    bts.Register(s)
    go func() {
        if err := s.Serve(lis); err != nil {
            log.Fatalf("failed to listen: %v", err)
        }
    }()
}

func dialer(context.Context, string) (net.Conn, error) {
    return lis.Dial()
}

func TestFileUpload(t *testing.T) {

    // Create the bytes transfer client

    conn, err := grpc.DialContext(context.Background(), "bufnet", grpc.WithContextDialer(dialer), grpc.WithInsecure())
    if err != nil {
        t.Fatalf("gRPC connect failed: %v", err)
    }
    defer conn.Close()
    client := CreateTransferClient(conn)

    t.Run("successful download", func(t *testing.T) {

        // Create a temp file and upload test

        content := make([]byte, 2048 * 1024 + 1024)

        tmpDir, err := ioutil.TempDir("", "test")
        if err != nil {
            t.Fatalf("failed to create temp remote directory: %v", err)
        }
        defer os.RemoveAll(tmpDir)

        srcPath := filepath.Join(tmpDir, "src_tmpfile")
        if err := ioutil.WriteFile(srcPath, content, 0666); err != nil {
            t.Fatalf("failed to create temp src file: %v", err)
        }

        uploadPath := filepath.Join(tmpDir, "upload_tempfile")

        // Create receiver/streamer

        fr := &receiver.FileReceiver{}
        bts.SetBytesReceiver(fr)

        fs := &streamer.FileStreamer{}

        // Perform upload

        if err := UploadBytes(client, context.Background(), srcPath, uploadPath, fs); err != nil {
            t.Fatalf("client failed: %v", err)
        }

        // Compare uploaded file to source

        srcf, err := ioutil.ReadFile(srcPath)
        if err != nil{
            t.Fatalf("failed to read src file: %v", err)
        }
        uploadf, err := ioutil.ReadFile(uploadPath)
        if err != nil{
            t.Fatalf("failed to read uploaded file: %v", err)
        }
        if !bytes.Equal(srcf, uploadf) {
            t.Fatalf("mismatch between remote and downloaded files")
        }
    })
}

func TestFileDownload(t *testing.T) {

    // Create the bytes transfer client

    conn, err := grpc.DialContext(context.Background(), "bufnet", grpc.WithContextDialer(dialer), grpc.WithInsecure())
    if err != nil {
        t.Fatalf("gRPC connect failed: %v", err)
    }
    defer conn.Close()
    client := CreateTransferClient(conn)

    t.Run("successful download", func(t *testing.T) {

        // Create a temp file and download dest

        content := make([]byte, 2048 * 1024 + 1024)

        tmpDir, err := ioutil.TempDir("", "test")
        if err != nil {
            t.Fatalf("failed to create temp remote directory: %v", err)
        }
        defer os.RemoveAll(tmpDir)

        remotePath := filepath.Join(tmpDir, "remote_tmpfile")
        if err := ioutil.WriteFile(remotePath, content, 0666); err != nil {
            t.Fatalf("failed to create temp remote file: %v", err)
        }

        dstPath := filepath.Join(tmpDir, "dst_tempfile")

        // Create receiver/streamer

        fs := &streamer.FileStreamer{}

        fr := &receiver.FileReceiver{}
        bts.SetBytesStreamer(fs)

        // Perform download

        if err := DownloadBytes(client, context.Background(), remotePath, dstPath, fr); err != nil {
            t.Fatalf("client failed: %v", err)
        }

        // Compare downloaded file to source

        remotef, err := ioutil.ReadFile(remotePath)
        if err != nil{
            t.Fatalf("failed to read remote file: %v", err)
        }
        downloadedf, err := ioutil.ReadFile(dstPath)
        if err != nil{
            t.Fatalf("failed to read downloaded file: %v", err)
        }
        if !bytes.Equal(remotef, downloadedf) {
            t.Fatalf("mismatch between remote and downloaded files")
        }
    })
}

