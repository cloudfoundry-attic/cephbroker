package utils

import (
	"code.cloudfoundry.org/goshims/ioutil"
	"code.cloudfoundry.org/goshims/os"
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

func ReadAndUnmarshal(object interface{}, dir string, fileName string, ioutil ioutilshim.Ioutil) error {
	path := dir + string(os.PathSeparator) + fileName

	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	err = json.Unmarshal(bytes, object)
	if err != nil {
		return err
	}

	return nil
}

func MarshalAndRecord(object interface{}, dir string, fileName string, osShim osshim.Os, ioutil ioutilshim.Ioutil) error {
	err := osShim.MkdirAll(dir, 0700)
	if err != nil {
		return err
	}

	path := dir + string(os.PathSeparator) + fileName

	bytes, err := json.MarshalIndent(object, "", " ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(path, bytes, 0700)
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

func Exists(path string, os osshim.Os) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}
	return true
}
