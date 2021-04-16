package download

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

	remotePath := filepath.Join(tmpDir, "remote_tmpfile")
	if err := ioutil.WriteFile(remotePath, content, 0666); err != nil {
        t.Fatalf("failed to create temp remote file: %v", err)
	}

    // Run server
    serverCtx, serverCtxCancel := context.WithTimeout(context.Background(), time.Second * 2)
    defer serverCtxCancel()
    server := NewServer(serverCtx, "127.0.0.1:50000")
    go func() {
        server.Start()
        defer server.Stop()
    }()

    // Run client
    clientCtx, clientCtxCancel := context.WithTimeout(context.Background(), time.Second * 2)
    defer clientCtxCancel()
    dstPath := filepath.Join(tmpDir, "dst_tempfile")
    if err := RunClient(clientCtx, "127.0.0.1:50000", remotePath, dstPath); err != nil {
        t.Fatalf("client failed: %v", err)
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

