package receiver

import (
    "os"
)

type FileReceiver struct {
    f *os.File
}

func (fr *FileReceiver) Init(path string) error {
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

