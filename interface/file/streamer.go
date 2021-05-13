package file

import (
    "os"
    "io"
    "fmt"

    "google.golang.org/protobuf/proto"
    "google.golang.org/protobuf/types/known/emptypb"

    pb "github.com/oren12321/gogrpcft/v2/interface/file/proto"
)

type FileStreamer struct {
    f *os.File
    empty bool
    buf []byte
    path string
}

func (fs *FileStreamer) Init(msg proto.Message) error {
    info, ok := msg.(*pb.File)
    if !ok {
        return fmt.Errorf("failed to convert 'Message' type to 'File' type")
    }
    fs.path = info.Path

    fs.buf = make([]byte, 2048)
    var err error
    fs.f, err = os.Open(fs.path)
    if err != nil {
        return fmt.Errorf("failed to open '%s': %v", fs.path, err)
    }

    return nil
}

func (fs *FileStreamer) HasNext() bool {
    return !fs.empty
}

func (fs *FileStreamer) GetNext() ([]byte, proto.Message, error) {
    n, err := fs.f.Read(fs.buf)
    if err == io.EOF {
        fs.empty = true
        return nil, &emptypb.Empty{}, nil
    }
    if err != nil {
        return nil, &emptypb.Empty{}, fmt.Errorf("failed to read data from '%s': %v", fs.path, err)
    }

    return fs.buf[:n], &emptypb.Empty{}, nil
}

func (fs *FileStreamer) Finalize() error {
    if fs.f != nil {
        if err := fs.f.Close(); err != nil {
            return fmt.Errorf("failed to close '%s': %v", fs.path, err)
        }
    }
    return nil
}

