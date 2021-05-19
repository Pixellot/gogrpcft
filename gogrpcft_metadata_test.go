package gogrpcft

import (
    "testing"
    "context"
    "fmt"

    "google.golang.org/grpc"

    "google.golang.org/protobuf/proto"
    "google.golang.org/protobuf/types/known/structpb"
    "google.golang.org/protobuf/types/known/emptypb"
)

type dummyStreamer struct {
    hasNext bool
}

func (ds *dummyStreamer) Init(msg proto.Message) error {
    ds.hasNext = true
    return nil
}

func (ds *dummyStreamer) HasNext() bool {
    if ds.hasNext {
        ds.hasNext = false
        return true
    }
    return false
}

func (ds *dummyStreamer) GetNext() ([]byte, proto.Message, error) {
    m, err := structpb.NewStruct(map[string]interface{}{
        "metadata": "info about dummy data",
    })
    if err != nil {
        return nil, nil, fmt.Errorf("failed to create 'structpb' type")
    }

    return []byte("dummy data"), m, nil
}

func (ds *dummyStreamer) Finalize() error {
    return nil
}

type dummyReceiver struct {}

func (dr *dummyReceiver) Init(msg proto.Message) error {
    return nil
}

func (dr *dummyReceiver) Push(data []byte, metadata proto.Message) error {
    if string(data) != "dummy data" {
        return fmt.Errorf("failed to receive data")
    }

    info, ok := metadata.(*structpb.Struct)
    if !ok {
        return fmt.Errorf("failed to receive data info")
    }

    fields := info.GetFields()
    if _, ok := fields["metadata"]; !ok {
        return fmt.Errorf("failed to get info key")
    }

    value := fields["metadata"]
    str := value.GetStringValue()
    if str != "info about dummy data" {
        return fmt.Errorf("received wrong data")
    }

    return nil
}

func (fr *dummyReceiver) Finalize() error {
    return nil
}

func TestUploadWithMetadata(t *testing.T) {

    // Create the bytes transfer client

    conn, err := grpc.DialContext(context.Background(), "bufnet", grpc.WithContextDialer(dialer), grpc.WithInsecure())
    if err != nil {
        t.Fatalf("gRPC connect failed: %v", err)
    }
    defer conn.Close()
    client := CreateTransferClient(conn)

    // Create receiver/streamer

    dr := &dummyReceiver{}
    bts.SetBytesReceiver(dr)

    ds := &dummyStreamer{}

    // Perform upload

    fromInfo := &emptypb.Empty{}
    toInfo := &emptypb.Empty{}

    if err := Send(client, context.Background(), fromInfo, toInfo, ds); err != nil {
        t.Fatalf("client failed: %v", err)
    }
}

func TestDownloadWithMetadata(t *testing.T) {

    // Create the bytes transfer client

    conn, err := grpc.DialContext(context.Background(), "bufnet", grpc.WithContextDialer(dialer), grpc.WithInsecure())
    if err != nil {
        t.Fatalf("gRPC connect failed: %v", err)
    }
    defer conn.Close()
    client := CreateTransferClient(conn)

    // Create receiver/streamer

    ds := &dummyStreamer{}

    dr := &dummyReceiver{}
    bts.SetBytesStreamer(ds)

    // Perform download

    fromInfo := &emptypb.Empty{}
    toInfo := &emptypb.Empty{}

    if err := Receive(client, context.Background(), fromInfo, toInfo, dr); err != nil {
        t.Fatalf("client failed: %v", err)
    }
}

