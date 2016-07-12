package utils

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
)

func UnmarshallDataFromRequest(r *http.Request, object interface{}) error {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}

	err = json.Unmarshal(body, object)
	if err != nil {
		return err
	}

	return nil
}

//go:generate counterfeiter -o ../cephfakes/fake_system_util.go . SystemUtil

type SystemUtil interface {
	MkdirAll(path string, perm os.FileMode) error
	WriteFile(filename string, data []byte, perm os.FileMode) error
	Remove(string) error
	Exists(path string) bool
	ReadFile(path string) ([]byte, error)
}

type realSystemUtil struct{}

func NewRealSystemUtil() SystemUtil {
	return &realSystemUtil{}
}

func (f *realSystemUtil) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (f *realSystemUtil) WriteFile(filename string, data []byte, perm os.FileMode) error {
	return ioutil.WriteFile(filename, data, perm)
}

func (f *realSystemUtil) Remove(path string) error {
	return os.RemoveAll(path)
}
func (f *realSystemUtil) Exists(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}
	return true
}
func (f *realSystemUtil) ReadFile(path string) ([]byte, error) {
	return readFile(path)
}

func ReadAndUnmarshal(object interface{}, dir string, fileName string, s SystemUtil) error {
	path := dir + string(os.PathSeparator) + fileName

	bytes, err := s.ReadFile(path)
	if err != nil {
		return err
	}

	err = json.Unmarshal(bytes, object)
	if err != nil {
		return err
	}

	return nil
}

func MarshalAndRecord(object interface{}, dir string, fileName string, s SystemUtil) error {
	err := s.MkdirAll(dir, 0700)
	if err != nil {
		return err
	}

	path := dir + string(os.PathSeparator) + fileName

	bytes, err := json.MarshalIndent(object, "", " ")
	if err != nil {
		return err
	}

	return s.WriteFile(path, bytes, 0700)
}

func readFile(path string) (content []byte, err error) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		return
	}
	content = bytes

	return
}

