package upload

import (
    "testing"
    "io/ioutil"
    "path/filepath"
    "context"
    "os"
    "time"
    "bytes"
)

func TestDownload(t *testing.T) {

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
    serverErrch := make(chan error)
    serverCtx, serverCtxCancel := context.WithTimeout(context.Background(), time.Second * 2)
    defer serverCtxCancel()
    server := NewServer(serverCtx, "127.0.0.1:50001")
    go func() {
        serverErrch <- server.Start()
    }()
    defer server.Stop()

    // Run client
    clientCtx, clientCtxCancel := context.WithTimeout(context.Background(), time.Second * 2)
    defer clientCtxCancel()
    uploadPath := filepath.Join(tmpDir, "upload_tempfile")
    if err := RunClient(clientCtx, "127.0.0.1:50001", srcPath, uploadPath); err != nil {
        t.Fatalf("client failed: %v", err)
    }

    if err := <-serverErrch; err != nil {
        t.Fatalf("server failed: %v", err)
    }

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

