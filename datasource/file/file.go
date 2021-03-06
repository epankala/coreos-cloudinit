package file

import (
	"io/ioutil"
	"os"
)

type localFile struct {
	path string
}

func NewDatasource(path string) *localFile {
	return &localFile{path}
}

func (f *localFile) IsAvailable() bool {
	_, err := os.Stat(f.path)
	return !os.IsNotExist(err)
}

func (f *localFile) AvailabilityChanges() bool {
	return true
}

func (f *localFile) ConfigRoot() string {
	return ""
}

func (f *localFile) FetchMetadata() ([]byte, error) {
	return []byte{}, nil
}

func (f *localFile) FetchUserdata() ([]byte, error) {
	return ioutil.ReadFile(f.path)
}

func (f *localFile) FetchNetworkConfig(filename string) ([]byte, error) {
	return nil, nil
}

func (f *localFile) Type() string {
	return "local-file"
}
