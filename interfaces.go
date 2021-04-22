package gogrpcft

type BytesStreamer interface {
    Init(msg string) error
    HasNext() bool
    GetNext() ([]byte, error)
    Finalize() error
}

type BytesReceiver interface {
    Init(msg string) error
    Push(data []byte) error
    Finalize() error
}

