package file

import (
	"bytes"
	"crypto/md5"
	"encoding/gob"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"gitlab.intra.wepiao.com/arch/wcache/core"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"time"
)

// FileCacheItem is basic unit of file cache adapter.
type FileCacheItem struct {
	Data       interface{}
	Lastaccess time.Time
	Expired    time.Time
}

// FileCache Config
var (
	FileCachePath           = "cache"     // cache directory
	FileCacheFileSuffix     = ".bin"      // cache file suffix
	FileCacheDirectoryLevel = 2           // cache file deep level if auto generated cache files.
	FileCacheEmbedExpiry    time.Duration // cache expire time, default is no expire forever.
)

// FileCache is cache adapter for file storage.
type Cache struct {
	CachePath      string
	FileSuffix     string
	DirectoryLevel int
	EmbedExpiry    int
}

// NewFileCache Create new file cache with no config.
func NewFileCache() core.Cache {
	return &Cache{}
}

// Start will start and begin gc for file cache.
func (fc *Cache) Start(config string) error {
	var cfg map[string]string
	json.Unmarshal([]byte(config), &cfg)
	if _, ok := cfg["CachePath"]; !ok {
		cfg["CachePath"] = FileCachePath
	}
	if _, ok := cfg["FileSuffix"]; !ok {
		cfg["FileSuffix"] = FileCacheFileSuffix
	}
	if _, ok := cfg["DirectoryLevel"]; !ok {
		cfg["DirectoryLevel"] = strconv.Itoa(FileCacheDirectoryLevel)
	}
	if _, ok := cfg["EmbedExpiry"]; !ok {
		cfg["EmbedExpiry"] = strconv.FormatInt(int64(FileCacheEmbedExpiry.Seconds()), 10)
	}
	fc.CachePath = cfg["CachePath"]
	fc.FileSuffix = cfg["FileSuffix"]
	fc.DirectoryLevel, _ = strconv.Atoi(cfg["DirectoryLevel"])
	fc.EmbedExpiry, _ = strconv.Atoi(cfg["EmbedExpiry"])

	fc.Init()
	return nil
}

// Init will make new dir for file cache if not exist.
func (fc *Cache) Init() {
	if ok, _ := exists(fc.CachePath); !ok { // todo : error handle
		_ = os.MkdirAll(fc.CachePath, os.ModePerm) // todo : error handle
	}
}

// get cached file name. it's md5 encoded.
func (fc *Cache) getCacheFileName(key string) string {
	m := md5.New()
	io.WriteString(m, key)
	keyMd5 := hex.EncodeToString(m.Sum(nil))
	cachePath := fc.CachePath
	switch fc.DirectoryLevel {
	case 2:
		cachePath = filepath.Join(cachePath, keyMd5[0:2], keyMd5[2:4])
	case 1:
		cachePath = filepath.Join(cachePath, keyMd5[0:2])
	}

	if ok, _ := exists(cachePath); !ok { // todo : error handle
		_ = os.MkdirAll(cachePath, os.ModePerm) // todo : error handle
	}

	return filepath.Join(cachePath, fmt.Sprintf("%s%s", keyMd5, fc.FileSuffix))
}

// Get value from file cache.
// if non-exist or expired, return empty string.
func (fc *Cache) Get(key string) interface{} {
	fileData, err := FileGetContents(fc.getCacheFileName(key))
	if err != nil {
		return ""
	}
	var to FileCacheItem
	GobDecode(fileData, &to)
	if to.Expired.Before(time.Now()) {
		return ""
	}
	return to.Data
}

// Get cache from redis.
func (rc *Cache) HGet(key string, filed string) interface{} {
	return errors.New("file no")
}

// GetMulti gets values from file cache.
func (fc *Cache) GetMulti(keys []string) []interface{} {
	var rc []interface{}
	for _, key := range keys {
		rc = append(rc, fc.Get(key))
	}
	return rc
}

// Put value into file cache.
func (fc *Cache) Put(key string, val interface{}, timeout time.Duration) error {
	gob.Register(val)

	item := FileCacheItem{Data: val}
	if timeout == FileCacheEmbedExpiry {
		item.Expired = time.Now().Add((86400 * 365 * 10) * time.Second) // ten years
	} else {
		item.Expired = time.Now().Add(timeout)
	}
	item.Lastaccess = time.Now()
	data, err := GobEncode(item)
	if err != nil {
		return err
	}
	return FilePutContents(fc.getCacheFileName(key), data)
}

// Delete file cache value.
func (fc *Cache) Delete(key string) error {
	filename := fc.getCacheFileName(key)
	if ok, _ := exists(filename); ok {
		return os.Remove(filename)
	}
	return nil
}

// Incr will increase cached int value.
func (fc *Cache) Incr(key string) (int, error) {
	data := fc.Get(key)
	var incr int
	if reflect.TypeOf(data).Name() != "int" {
		incr = 0
	} else {
		incr = data.(int) + 1
	}
	fc.Put(key, incr, FileCacheEmbedExpiry)
	return incr, nil
}

// Decr will decrease cached int value.
func (fc *Cache) Decr(key string) (int, error) {
	data := fc.Get(key)
	var decr int
	if reflect.TypeOf(data).Name() != "int" || data.(int)-1 <= 0 {
		decr = 0
	} else {
		decr = data.(int) - 1
	}
	fc.Put(key, decr, FileCacheEmbedExpiry)
	return decr, nil
}

// IncrBy will increase cached int value.
func (fc *Cache) IncrBy(key string, num int) (int, error) {
	data := fc.Get(key)
	var incr int
	if reflect.TypeOf(data).Name() != "int" {
		incr = 0
	} else {
		incr = data.(int) + num
	}
	fc.Put(key, incr, FileCacheEmbedExpiry)
	return incr, nil
}

// DecrBy will decrease cached int value.
func (fc *Cache) DecrBy(key string, num int) (int, error) {
	data := fc.Get(key)
	var decr int
	if reflect.TypeOf(data).Name() != "int" || data.(int)-1 <= 0 {
		decr = 0
	} else {
		decr = data.(int) - num
	}
	fc.Put(key, decr, FileCacheEmbedExpiry)
	return decr, nil
}

// Push push
func (fc *Cache) Push(key string, val interface{}) error {
	return errors.New("file no")
}

// Pop pop
func (fc *Cache) Pop(key string) (interface{}, error) {
	return nil, errors.New("file no")
}

// IsExist check value is exist.
func (fc *Cache) IsExist(key string) bool {
	ret, _ := exists(fc.getCacheFileName(key))
	return ret
}

// ClearAll will clean cached files.
// not implemented.
func (fc *Cache) ClearAll() error {
	return nil
}

// check file exist.
func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// FileGetContents Get bytes to file.
// if non-exist, create this file.
func FileGetContents(filename string) (data []byte, e error) {
	f, e := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, os.ModePerm)
	if e != nil {
		return
	}
	defer f.Close()
	stat, e := f.Stat()
	if e != nil {
		return
	}
	data = make([]byte, stat.Size())
	result, e := f.Read(data)
	if e != nil || int64(result) != stat.Size() {
		return nil, e
	}
	return
}

// FilePutContents Put bytes to file.
func FilePutContents(filename string, content []byte) error {
	fp, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		return err
	}
	defer fp.Close()
	_, err = fp.Write(content)
	return err
}

// GobEncode Gob encodes file cache item.
func GobEncode(data interface{}) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	enc := gob.NewEncoder(buf)
	err := enc.Encode(data)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), err
}

// GobDecode Gob decodes file cache item.
func GobDecode(data []byte, to *FileCacheItem) error {
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	return dec.Decode(&to)
}

//func init() {
//	wcache.Register("file", NewFileCache)
//}
