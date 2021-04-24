package file

import (
    "os"
    "io"

    "google.golang.org/protobuf/proto"
    pb "github.com/oren12321/gogrpcft/v2/interface/file/proto"
)

type FileStreamer struct {
    f *os.File
    empty bool
    buf []byte
}

func (fs *FileStreamer) Init(msg proto.Message) error {
    info := msg.(*pb.Info)
    path := info.Path

    fs.buf = make([]byte, 2048)
    var err error
    fs.f, err = os.Open(path)
    return err
}

func (fs *FileStreamer) HasNext() bool {
    return !fs.empty
}

func (fs *FileStreamer) GetNext() ([]byte, error) {
    n, err := fs.f.Read(fs.buf)
    if err == io.EOF {
        fs.empty = true
        return nil, nil
    }
    if err != nil {
        return nil, err
    }
    return fs.buf[:n], nil
}

func (fs *FileStreamer) Finalize() error {
    if fs.f != nil {
        if err := fs.f.Close(); err != nil {
            return err
        }
    }
    return nil
}

