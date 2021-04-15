package gogrpcft

import (
    "testing"
    "io/ioutil"
    "path/filepath"
    "context"
    "os"
    "time"
    "bytes"
    "net"

    "google.golang.org/grpc"
)

func TestDownload(t *testing.T) {

    // Create a temp file for the test
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

    // Run server

    lis, err := net.Listen("tcp", "127.0.0.1:50000")
    if err != nil {
        t.Fatalf("Failed to listen: %v", err)
    }

    server := grpc.NewServer()
    RegisterFilesTransferServer(server)

    //serverCtx, serverCtxCancel := context.WithTimeout(context.Background(), time.Second * 2)
    //defer serverCtxCancel()
    //server := NewServer(serverCtx, "127.0.0.1:50000")
    go func() {
        server.Serve(lis)
    }()
    defer server.GracefulStop()

    conn, err := grpc.Dial("127.0.0.1:50000", grpc.WithInsecure(), grpc.WithBlock()/*, grpc.WithTimeout(time.Second)*/)
    if err != nil {
        t.Fatalf("gRPC connect failed: %v", err)
    }
    defer conn.Close()

    // Run client
    client := CreateFilesTransferClient(conn)

    dstPath := filepath.Join(tmpDir, "dst_tempfile")
    for i := 0; i < 10; i++ {
    clientCtx, clientCtxCancel := context.WithTimeout(context.Background(), time.Second * 2)
    defer clientCtxCancel()
    if err := DownloadFile(client, clientCtx, remotePath, dstPath); err != nil {
        t.Fatalf("client failed: %v", err)
    }
}

    // Compare remote and downloaded files
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
}

