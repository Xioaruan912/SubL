package middlewares

import (
	"compress/gzip"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

// StaticFS 自定义静态资源服务：支持 gzip，正确处理 Content-Encoding。
// 替代 gin 的 r.StaticFS（后者用 http.ServeContent，会覆盖 Content-Length 且 gzip 难控制）。
func StaticFS(root fs.FS) gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		if !strings.HasPrefix(path, "/static/") {
			c.Next()
			return
		}
		name := strings.TrimPrefix(path, "/static/")
		file, err := root.Open(name)
		if err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		defer file.Close()

		info, err := file.Stat()
		if err != nil || info.IsDir() {
			c.Status(http.StatusNotFound)
			return
		}

		// 设置 Content-Type（基于扩展名）
		ct := mime.TypeByExtension(filepath.Ext(name))
		if ct == "" {
			ct = "application/octet-stream"
		}
		c.Header("Content-Type", ct)

		// 判断是否可压缩 + 客户端是否接受 gzip
		acceptsGzip := strings.Contains(c.Request.Header.Get("Accept-Encoding"), "gzip")
		if acceptsGzip && isCompressible(ct) {
			c.Header("Content-Encoding", "gzip")
			c.Header("Cache-Control", "public, max-age=31536000, immutable")
			c.Status(http.StatusOK)
			gz := gzip.NewWriter(c.Writer)
			defer gz.Close()
			_, err = io.Copy(gz, file)
			if err != nil {
				return
			}
			return
		}

		c.Header("Content-Length", itoa64(info.Size()))
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
		c.Status(http.StatusOK)
		io.Copy(c.Writer, file)
	}
}

func ioCopy(dst io.Writer, src io.Reader) (int64, error) {
	return io.Copy(dst, src)
}

func isCompressible(ct string) bool {
	ct = strings.ToLower(ct)
	return strings.Contains(ct, "text/") ||
		strings.Contains(ct, "application/json") ||
		strings.Contains(ct, "application/javascript") ||
		strings.Contains(ct, "application/x-javascript") ||
		strings.Contains(ct, "application/xml") ||
		strings.Contains(ct, "image/svg+xml") ||
		strings.Contains(ct, "font/") ||
		strings.Contains(ct, "application/wasm")
}

func itoa64(n int64) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}