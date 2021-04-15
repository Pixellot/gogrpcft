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

func TestUpload(t *testing.T) {

    // Create a temp file for the test
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

    // Run server

    lis, err := net.Listen("tcp", "127.0.0.1:50001")
    if err != nil {
        t.Fatalf("Failed to listen: %v", err)
    }

    server := grpc.NewServer()
    RegisterFilesTransferServer(server)

    //serverErrch := make(chan error)
    //serverCtx, serverCtxCancel := context.WithTimeout(context.Background(), time.Second * 2)
    //defer serverCtxCancel()
    //server := NewServer(serverCtx, "127.0.0.1:50001")
    go func() {
        server.Serve(lis)
    }()
    defer server.GracefulStop()

    // Run client

    conn, err := grpc.Dial("127.0.0.1:50001", grpc.WithInsecure(), grpc.WithBlock()/*, grpc.WithTimeout(time.Second)*/)
    if err != nil {
        t.Fatalf("gRPC connect failed: %v", err)
    }
    defer conn.Close()

    client := CreateFilesTransferClient(conn)

    uploadPath := filepath.Join(tmpDir, "upload_tempfile")
    for i := 0; i < 10; i++ {
    clientCtx, clientCtxCancel := context.WithTimeout(context.Background(), time.Second * 2)
    defer clientCtxCancel()
    if err := UploadFile(client, clientCtx, srcPath, uploadPath); err != nil {
        t.Fatalf("client failed: %v", err)
    }
}

    /*if err := <-serverErrch; err != nil {
        t.Fatalf("server failed: %v", err)
    }*/

    // Compare remote and downloaded files
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
}

