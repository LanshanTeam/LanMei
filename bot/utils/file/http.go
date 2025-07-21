package file

import (
	"io"
	"net/http"
	"os"
	"strings"
)

func FileStorageHandler(w http.ResponseWriter, r *http.Request) {
	// 获取 URL 参数，例如 /v1/files/wcloud.png
	path := r.URL.Path
	filename := strings.TrimPrefix(path, "/v1/file/")
	if filename == "" {
		http.Error(w, "文件名不能为空", http.StatusBadRequest)
		return
	}

	filePath := "./data/wcloud/" + filename
	defer os.Remove(filePath)

	file, err := os.Open(filePath)
	if err != nil {
		http.Error(w, "文件未找到", http.StatusNotFound)
		return
	}
	defer file.Close()

	buffer := make([]byte, 512)
	n, _ := file.Read(buffer)
	contentType := http.DetectContentType(buffer[:n])

	file.Seek(0, io.SeekStart)

	w.Header().Set("Content-Type", contentType)

	io.Copy(w, file)
}
