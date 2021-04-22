package gogrpcft

import (
    "testing"
    "log"
    "context"
    "net"
    "bytes"

    "google.golang.org/grpc"
    "google.golang.org/grpc/test/bufconn"
)

var btlis *bufconn.Listener
var bts *BytesTransferServer

func init() {
    btlis = bufconn.Listen(1024 * 1024)
    s := grpc.NewServer()
    bts = &BytesTransferServer{}
    bts.Register(s)
    go func() {
        if err := s.Serve(btlis); err != nil {
            log.Fatalf("failed to listen: %v", err)
        }
    }()
}

func btdialer(context.Context, string) (net.Conn, error) {
    return btlis.Dial()
}

func TestUploadBytes(t *testing.T) {

    // Create the bytes transfer client

    conn, err := grpc.DialContext(context.Background(), "bufnet", grpc.WithContextDialer(btdialer), grpc.WithInsecure())
    if err != nil {
        t.Fatalf("gRPC connect failed: %v", err)
    }
    defer conn.Close()
    client := CreateTransferClient(conn)

    t.Run("successful download", func(t *testing.T) {

        // Create receiver/streamer

        content := make([]byte, 2048 * 1024 + 1024)

        receiver := &simpleBufferReceiver{}
        bts.SetBytesReceiver(receiver)

        streamer := &simpleBufferStreamer{buffer: content}

        // Perform upload

        if err := UploadBytes(client, context.Background(), "<unused>", "<unused>", streamer); err != nil {
            t.Fatalf("client failed: %v", err)
        }

        // Compare uploaded file to source

        if !bytes.Equal(content, receiver.buffer) {
            t.Fatalf("mismatch between remote and downloaded files")
        }
    })
}

func TestDownloadBytes(t *testing.T) {

    // Create the bytes transfer client

    conn, err := grpc.DialContext(context.Background(), "bufnet", grpc.WithContextDialer(btdialer), grpc.WithInsecure())
    if err != nil {
        t.Fatalf("gRPC connect failed: %v", err)
    }
    defer conn.Close()
    client := CreateTransferClient(conn)

    t.Run("successful download", func(t *testing.T) {

        // Create receiver/streamer

        content := make([]byte, 2048 * 1024 + 1024)

        receiver := &simpleBufferReceiver{}

        streamer := &simpleBufferStreamer{buffer: content}
        bts.SetBytesStreamer(streamer)

        // Perform download

        if err := DownloadBytes(client, context.Background(), "<unused>", "<unused>", receiver); err != nil {
            t.Fatalf("client failed: %v", err)
        }

        // Compare downloaded bytes to source

        if !bytes.Equal(content, receiver.buffer) {
            t.Fatalf("mismatch between remote and downloaded files")
        }
    })
}


type simpleBufferStreamer struct {
    buffer []byte
    index int
}

func (sbs *simpleBufferStreamer) Init(msg string) error {
    sbs.index = 0
    return nil
}

func (sbs *simpleBufferStreamer) HasNext() bool {
    return sbs.index < len(sbs.buffer)
}

func (sbs *simpleBufferStreamer) GetNext() ([]byte, error) {
    if sbs.index + 1234 - 1 < len(sbs.buffer) {
        data := sbs.buffer[sbs.index:(sbs.index + 1234)]
        sbs.index += 1234
        return data, nil
    }
    reminder := len(sbs.buffer) - sbs.index
    data := sbs.buffer[sbs.index:(sbs.index + reminder)]
    sbs.index += reminder
    return data, nil
}

func (sbs *simpleBufferStreamer) Finalize() error {
    return nil
}


type simpleBufferReceiver struct {
    buffer []byte
}

func (sbr *simpleBufferReceiver) Init(msg string) error {
    sbr.buffer = make([]byte, 0)
    return nil
}

func (sbr *simpleBufferReceiver) Push(data []byte) error {
    sbr.buffer = append(sbr.buffer, data...)
    return nil
}

func (sbr *simpleBufferReceiver) Finalize() error {
    return nil
}

