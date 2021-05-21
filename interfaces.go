package gogrpcft

import (
    "google.golang.org/protobuf/proto"
)

// BytesStreamer represents an interface for bytes streaming.
type BytesStreamer interface {

    // Init initialize the streamer.
    Init(msg proto.Message) error

    // HasNext returns true if there are more bytes to stream.
    HasNext() bool

    // GetNext returns the next bytes with optional additional info.
    GetNext() ([]byte, proto.Message, error)

    // Finalize clear resources of the streamer.
    Finalize() error
}


// BytesReceiver represetns an interface for bytes reception.
type BytesReceiver interface {

    // Init initialize the receiver.
    Init(msg proto.Message) error

    // Push processing the received bytes and their optional additional info.
    Push(data []byte, metadata proto.Message) error

    // Finalize clear resources of the receiver.
    Finalize() error
}

