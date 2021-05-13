package file

import (
    "os"
    "fmt"

    "google.golang.org/protobuf/proto"
    pb "github.com/oren12321/gogrpcft/v2/interface/file/proto"
)

type FileReceiver struct {
    f *os.File
    path string
}

func (fr *FileReceiver) Init(msg proto.Message) error {
    info, ok := msg.(*pb.File)
    if !ok {
        return fmt.Errorf("failed to convert 'Message' type to 'File' type")
    }
    fr.path = info.Path

    var err error
    fr.f, err = os.Create(fr.path)
    if err != nil {
        return fmt.Errorf("failed to create '%s': %v", fr.path, err)
    }

    return nil
}

func (fr *FileReceiver) Push(data []byte, metadata proto.Message) error {
    _, err := fr.f.Write(data)
    if err != nil {
        return fmt.Errorf("failed to write data to '%s' - file may be corrupted: %v", fr.path, err)
    }

    return nil
}

func (fr *FileReceiver) Finalize() error {
    if fr.f != nil {
        if err := fr.f.Close(); err != nil {
            return fmt.Errorf("failed to close '%s': %v", fr.path, err)
        }
    }

    return nil
}

