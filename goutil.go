package goutil

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"math/big"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/ahui2016/goutil/graphics"
)

const (
	// ISO8601 需要根据服务器的具体时区来设定正确的时区
	ISO8601 = "2006-01-02T15:04:05.999+08:00"
)

// TimeNow .
func TimeNow() string {
	return time.Now().Format(ISO8601)
}

// NewID 返回一个上升趋势的随机 id, 由时间戳与随机数组成。
// 时间戳确保其上升趋势（大致有序），随机数确保其随机性（防止被穷举）。
// NewID 考虑了 “生成新 id 的速度”、 “并发生成时防止冲突” 与 “id 长度”
// 这三者的平衡，适用于大多数中、小规模系统（当然，不适用于大型系统）。
func NewID() string {
	var max int64 = 100_000_000
	n, err := rand.Int(rand.Reader, big.NewInt(max))
	if err != nil {
		panic(err)
	}
	timestamp := time.Now().Unix()
	idInt64 := timestamp*max + n.Int64()
	return strconv.FormatInt(idInt64, 36)
}

// PathIsNotExist .
func PathIsNotExist(name string) bool {
	_, err := os.Lstat(name)
	if os.IsNotExist(err) {
		return true
	}
	if err != nil {
		panic(err)
	}
	return false
}

// PathIsExist .
func PathIsExist(name string) bool {
	return !PathIsNotExist(name)
}

// MustMkdir 确保有一个名为 dirName 的文件夹，
// 如果没有则自动创建，如果已存在则不进行任何操作。
func MustMkdir(dirName string) {
	if PathIsNotExist(dirName) {
		if err := os.Mkdir(dirName, 0600); err != nil {
			panic(err)
		}
	}
}

// Base64Encode .
func Base64Encode(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

// Base64Decode .
func Base64Decode(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}

// GetFilesByExt .
func GetFilesByExt(dir, ext string) ([]string, error) {
	pattern := filepath.Join(dir, "*"+ext)
	filePaths, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}
	return filePaths, nil
}

// GetID checks if the r.FormValue("id") is empty or not,
// if it is empty, return false; if it is not empty,
// return the id and true.
func GetID(w http.ResponseWriter, r *http.Request) (string, bool) {
	id := r.FormValue("id")
	if id == "" {
		JsonMessage(w, "id is empty", 400)
		return "", false
	}
	return id, true
}

// CheckErr 检查 err, 如果有错就以 json 形式返回给前端，并返回 true.
// 如果没有错误则返回 false.
func CheckErr(w http.ResponseWriter, err error, code int) bool {
	if err != nil {
		log.Println(err)
		JsonMessage(w, err.Error(), code)
		return true
	}
	return false
}

func JsonMsgOK(w http.ResponseWriter) {
	JsonMessage(w, "OK", 200)
}

func JsonMsg404(w http.ResponseWriter) {
	JsonMessage(w, "Not Found", 404)
}

func JsonRequireLogin(w http.ResponseWriter) {
	JsonMessage(w, "Require Login", http.StatusUnauthorized)
}

// JsonMessage 主要用于向前端返回出错消息。
func JsonMessage(w http.ResponseWriter, message string, code int) {
	msg := map[string]string{"message": message}
	JsonResponse(w, msg, code)
}

// JsonResponse 要用于向前端返回有用数据。
// 参考 https://stackoverflow.com/questions/59763852/can-you-return-json-in-golang-http-error
func JsonResponse(w http.ResponseWriter, obj interface{}, code int) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(obj)
}

// GetFileContents gets contents from http.Request.FormFile("file").
// It also verifies the file has not been corrupted.
func GetFileContents(r *http.Request) ([]byte, error) {
	file, _, err := r.FormFile("file")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// 将文件内容全部读入内存
	contents, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}

	// 根据文件内容生成 checksum 并检查其是否正确
	if Sha256Hex(contents) != r.FormValue("checksum") {
		return nil, errors.New("checksums do not match")
	}
	return contents, nil
}

// Sha256Hex .
func Sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// CreateFile 把 src 的数据写入 filePath, 权限是 0600, 自动关闭 file.
func CreateFile(filePath string, src io.Reader) error {
	_, file, err := CreateReturnFile(filePath, src)
	if err == nil {
		file.Close()
	}
	return err
}

// CreateReturnFile 把 src 的数据写入 filePath, 权限是 0600,
// 会自动创建或覆盖文件，返回 file, 要记得关闭资源。
func CreateReturnFile(filePath string, src io.Reader) (int64, *os.File, error) {
	f, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return 0, nil, err
	}
	size, err := io.Copy(f, src)
	if err != nil {
		return 0, nil, err
	}
	return size, f, nil
}

// BytesToThumb creates a thumbnail from img, uses default size and default quality,
// and write the thumbnail to thumbPath.
func BytesToThumb(img []byte, thumbPath string) error {
	buf, err := graphics.Thumbnail(img, 0, 0)
	if err != nil {
		return err
	}
	return CreateFile(thumbPath, buf)
}

// TypeByFilename 从文件名中截取后缀名，然后判断文件类型。
func TypeByFilename(filename string) string {
	return mime.TypeByExtension(filepath.Ext(filename))
}
