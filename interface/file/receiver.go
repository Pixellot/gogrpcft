package file

import (
    "os"

    "google.golang.org/protobuf/proto"
    pb "github.com/oren12321/gogrpcft/v2/interface/file/proto"
)

type FileReceiver struct {
    f *os.File
}

func (fr *FileReceiver) Init(msg proto.Message) error {
    info := msg.(*pb.File)
    path := info.Path

    var err error
    fr.f, err = os.Create(path)
    return err
}

func (fr *FileReceiver) Push(data []byte) error {
    _, err := fr.f.Write(data)
    return err
}

func (fr *FileReceiver) Finalize() error {
    if fr.f != nil {
        if err := fr.f.Close(); err != nil {
            return err
        }
    }
    return nil
}

