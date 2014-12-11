package appendstorage

import (
	"bytes"
	"encoding/gob"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strconv"

	"github.com/minio-io/minio/pkgs/storage"
)

type appendStorage struct {
	RootDir     string
	file        *os.File
	objects     map[string]Header
	objectsFile string
}

type Header struct {
	Path   string
	Offset int64
	Length int
	Crc    uint32
}

func NewStorage(rootDir string, slice int) (storage.ObjectStorage, error) {
	rootPath := path.Join(rootDir, strconv.Itoa(slice))
	// TODO verify and fix partial writes
	file, err := os.OpenFile(rootPath, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return &appendStorage{}, err
	}
	objectsFile := path.Join(rootDir, strconv.Itoa(slice)+".map")
	objects := make(map[string]Header)
	if _, err := os.Stat(objectsFile); err == nil {
		mapFile, err := os.Open(objectsFile)
		defer mapFile.Close()
		if err != nil {
			return &appendStorage{}, nil
		}
		dec := gob.NewDecoder(mapFile)
		err = dec.Decode(&objects)
		if err != nil && err != io.EOF {
			return &appendStorage{}, nil
		}
	}
	if err != nil {
		return &appendStorage{}, err
	}
	return &appendStorage{
		RootDir:     rootDir,
		file:        file,
		objects:     objects,
		objectsFile: objectsFile,
	}, nil
}

func (storage *appendStorage) Get(objectPath string) (io.Reader, error) {
	header, ok := storage.objects[objectPath]
	if ok == false {
		return nil, nil
	}

	offset := header.Offset
	length := header.Length

	object := make([]byte, length)
	_, err := storage.file.ReadAt(object, offset)
	if err != nil {
		return nil, err
	}
	return bytes.NewBuffer(object), nil
}

func (aStorage *appendStorage) Put(objectPath string, object io.Reader) error {
	header := Header{
		Path:   objectPath,
		Offset: 0,
		Length: 0,
		Crc:    0,
	}
	offset, err := aStorage.file.Seek(0, os.SEEK_END)
	if err != nil {
		return err
	}
	objectBytes, err := ioutil.ReadAll(object)
	if err != nil {
		return err
	}
	if _, err := aStorage.file.Write(objectBytes); err != nil {
		return err
	}
	header.Offset = offset
	header.Length = len(objectBytes)
	aStorage.objects[objectPath] = header
	var mapBuffer bytes.Buffer
	encoder := gob.NewEncoder(&mapBuffer)
	encoder.Encode(aStorage.objects)
	ioutil.WriteFile(aStorage.objectsFile, mapBuffer.Bytes(), 0600)
	return nil
}

func (aStorage *appendStorage) List(listPath string) ([]storage.ObjectDescription, error) {
	return nil, errors.New("Not Implemented")
}