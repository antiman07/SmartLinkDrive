package main

import (
	"flag"
	"fmt"
	"net/http"
	"time"
)

// 说明：
// 第一阶段规划里提到 “Kong/Nginx + gRPC-Gateway”。
// 当前仓库还没有业务 proto（只有 health），因此这里先提供一个最小可运行的 HTTP 入口骨架：
// - /healthz: 网关自身健康检查
// 后续接入 grpc-gateway 时：
// 1) 在 internal/api/proto 下补齐业务 proto，并添加 google.api.http 注解
// 2) 用 Makefile 中的 protoc 生成 gateway handlers
// 3) 在这里初始化 grpc-gateway mux，把 HTTP 映射到后端 gRPC（并可配合 Consul 解析）

var (
	listenAddr = flag.String("listen", ":8080", "HTTP listen address")
)

func main() {
	flag.Parse()

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("ok"))
	})

	srv := &http.Server{
		Addr:              *listenAddr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	fmt.Printf("api-gateway listening on %s\n", *listenAddr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		panic(err)
	}
}

