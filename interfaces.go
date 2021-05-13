package gogrpcft

import (
    "google.golang.org/protobuf/proto"
)

type BytesStreamer interface {
    Init(msg proto.Message) error
    HasNext() bool
    GetNext() ([]byte, proto.Message, error)
    Finalize() error
}

type BytesReceiver interface {
    Init(msg proto.Message) error
    Push(data []byte, metadata proto.Message) error
    Finalize() error
}

