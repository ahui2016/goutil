package goutil

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"log"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

const (
	// ISO8601 需要根据服务器的具体时区来设定正确的时区
	ISO8601 = "2006-01-02T15:04:05.999+08:00"
)

// TimeNow .
func TimeNow() string {
	return time.Now().Format(ISO8601)
}

// NewID .
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

// FilesInDir .
func FilesInDir(dir, ext string) ([]string, error) {
	pattern := filepath.Join(dir, "*"+ext)
	filePaths, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}
	return filePaths, nil
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
