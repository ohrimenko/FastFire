package filecache

import (
	"bytes"
	"encoding/gob"
	"errors"
	"io/ioutil"

	"crypto/md5"
	"encoding/hex"
	"os"
	"path/filepath"
	"time"

	"encoding/base64"
)

type CacheFile struct {
	cachedir string
	valid    bool
}

type DataCache struct {
	Remove int64
	Value  interface{}
}

func NewCacheFile(cachedir string) (*CacheFile, error) {
	if cachedir == "" {
		return nil, errors.New("cachedir not exist")
	}

	if !is_dir(cachedir) {
		err := os.MkdirAll(cachedir, os.ModePerm)

		if err != nil {
			return nil, errors.New("cachedir error create")
		}
	}

	cf := &CacheFile{}

	cf.cachedir = cachedir
	cf.valid = true

	go func() {
		for {
			cf.ClearTrash()
			<-time.After(time.Hour * 1)
		}
	}()

	return cf, nil
}

func (cf *CacheFile) Get(key interface{}) (interface{}, error) {
	path := cf.getPath(key)

	if is_file(path) {
		body, err := os.ReadFile(path)

		if err == nil {
			data, err := unserialize(body)

			if err == nil {
				if data.Remove > 0 && data.Remove < time.Now().Unix() {
					cf.Del(key)
				} else {
					return data.Value, nil
				}
			} else {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	return nil, errors.New("not value by key")
}

func (cf *CacheFile) Set(key interface{}, value interface{}, interval int64) bool {
	path := cf.getPath(key)

	dir := filepath.Dir(path)

	if !is_dir(dir) {
		err := os.MkdirAll(dir, os.ModePerm)

		if err != nil {
			return false
		}
	}

	dc := DataCache{
		Remove: 0,
		Value:  value,
	}

	if interval > 0 {
		dc.Remove = time.Now().Unix() + interval
	}

	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0777)

	if err == nil {
		defer file.Close()

		ds, err := serialize(dc)
		if err == nil {
			file.Write(ds)
			return true
		}
	}

	return false
}

func (cf *CacheFile) Del(key interface{}) {
	path := cf.getPath(key)
	dir := filepath.Dir(path)

	os.Remove(path)

	files, err := ioutil.ReadDir(dir)
	if err == nil {
		if len(files) == 0 {
			os.Remove(dir)

			dir = filepath.Dir(dir)

			files, err := ioutil.ReadDir(dir)
			if err == nil {
				if len(files) == 0 {
					os.Remove(dir)
				}
			}
		}
	}
}

func (cf *CacheFile) Clear() {
	files, err := ioutil.ReadDir(cf.cachedir)
	if err == nil {
		for _, file := range files {
			os.RemoveAll(cf.cachedir + "/" + file.Name())
		}
	}
}

func (cf *CacheFile) ClearTrash() {
	files, err := ioutil.ReadDir(cf.cachedir)
	if err == nil {
		for _, file := range files {
			cf.recursiveTrash(cf.cachedir + "/" + file.Name())
		}
	}
}

func (cf *CacheFile) recursiveTrash(path string) {
	if is_dir(path) {
		files, err := ioutil.ReadDir(path)
		if err == nil {
			for _, file := range files {
				cf.recursiveTrash(path + "/" + file.Name())
			}

			files, err := ioutil.ReadDir(path)
			if err == nil {
				if len(files) == 0 {
					os.Remove(path)
				}
			}
		}
	} else {
		body, err := os.ReadFile(path)

		if err == nil {
			data, err := unserialize(body)

			if err == nil {
				if data.Remove > 0 && data.Remove < time.Now().Unix() {
					os.Remove(path)
				}
			} else {
				os.Remove(path)
			}
		}
	}
}

func (cf *CacheFile) getPath(key interface{}) string {
	key_str := "/"

	buf := new(bytes.Buffer)
	enc := gob.NewEncoder(buf)
	err := enc.Encode(key)
	if err == nil {
		key_str = base64.StdEncoding.EncodeToString([]byte(buf.Bytes()))
	}

	hash := md5Hash(key_str)

	return cf.cachedir + "/" + "." + substr(hash, 0, 2) + "/" + "." + substr(hash, 2, 2) + "/" + "." + hash
}

func is_exist(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}

	return true
}

func is_dir(path string) bool {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false
	}

	if os.IsNotExist(err) {
		return false
	}

	return fileInfo.IsDir()
}

func is_file(path string) bool {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false
	}

	if os.IsNotExist(err) {
		return false
	}

	return !fileInfo.IsDir()
}

func md5Hash(text string) string {
	hasher := md5.New()
	hasher.Write([]byte(text))
	return hex.EncodeToString(hasher.Sum(nil))
}

func substr(input string, start int, length int) string {
	asRunes := []rune(input)

	if start >= len(asRunes) {
		return ""
	}

	if start+length > len(asRunes) {
		length = len(asRunes) - start
	}

	return string(asRunes[start : start+length])
}

func serialize(data DataCache) ([]byte, error) {
	buf := new(bytes.Buffer)
	enc := gob.NewEncoder(buf)
	err := enc.Encode(data)
	if err == nil {
		return buf.Bytes(), nil
	}
	return nil, err
}

func unserialize(data []byte) (DataCache, error) {
	dec := gob.NewDecoder(bytes.NewBuffer(data))
	dc := &DataCache{}
	err := dec.Decode(dc)
	return *dc, err
}
