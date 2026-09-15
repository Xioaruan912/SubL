package utils

import (
	"log"
	"runtime/debug"
)

// RecoverPanic 用于 defer，捕获后台 goroutine 中的 panic，避免单个异常导致进程退出。
func RecoverPanic(name string) {
	if r := recover(); r != nil {
		log.Printf("[panic] %s: %v\n%s", name, r, debug.Stack())
	}
}

// SafeGo 在独立 goroutine 中运行 fn，并捕获 panic。
func SafeGo(name string, fn func()) {
	go func() {
		defer RecoverPanic(name)
		fn()
	}()
}
