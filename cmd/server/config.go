package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

type config struct {
	addr             string
	databasePath     string
	issuer           string
	secret           string
	selfcheck        bool
	selfcheckTimeout time.Duration
}

func parseConfig(args []string) (config, error) {
	defaults, err := defaultAddress(os.Getenv("PORT"))
	if err != nil {
		return config{}, err
	}
	set := flag.NewFlagSet("caption-release-gate", flag.ContinueOnError)
	var value config
	set.StringVar(&value.addr, "addr", defaults, "HTTP 监听地址，仅允许回环 IP")
	set.StringVar(&value.databasePath, "db", "caption-release-gate.db", "SQLite 数据库文件")
	set.StringVar(&value.issuer, "issuer", "caption-gate-local-v1", "清单签发标识")
	set.StringVar(&value.secret, "signing-secret", "local-caption-gate-secret-2026", "本地清单签发密钥")
	set.BoolVar(&value.selfcheck, "selfcheck", false, "通过真实 HTTP 监听执行有界完整流程并退出")
	set.DurationVar(&value.selfcheckTimeout, "selfcheck-timeout", 20*time.Second, "自检总超时")
	if err := set.Parse(args); err != nil {
		return config{}, err
	}
	if set.NArg() != 0 {
		return config{}, fmt.Errorf("不支持位置参数: %s", strings.Join(set.Args(), " "))
	}
	if err := validateAddress(value.addr); err != nil {
		return config{}, err
	}
	if value.selfcheckTimeout <= 0 || value.selfcheckTimeout > 2*time.Minute {
		return config{}, fmt.Errorf("selfcheck-timeout 必须在 0 到 2 分钟之间")
	}
	if len(value.secret) < 16 {
		return config{}, fmt.Errorf("signing-secret 至少需要 16 字节")
	}
	if value.selfcheck {
		value.databasePath = ":memory:"
	}
	return value, nil
}

func defaultAddress(portValue string) (string, error) {
	if strings.TrimSpace(portValue) == "" {
		return "127.0.0.1:19081", nil
	}
	if portValue != strings.TrimSpace(portValue) {
		return "", fmt.Errorf("PORT 必须为纯端口号")
	}
	port, err := strconv.Atoi(portValue)
	if err != nil || port < 1024 || port > 65535 {
		return "", fmt.Errorf("PORT 必须是 1024 到 65535 之间的纯端口号")
	}
	return net.JoinHostPort("127.0.0.1", strconv.Itoa(port)), nil
}

func validateAddress(address string) error {
	host, portText, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("addr 必须是 host:port: %w", err)
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return fmt.Errorf("addr 必须使用明确的回环 IP，禁止 %q", host)
	}
	port, err := strconv.Atoi(portText)
	if err != nil || port < 1024 || port > 65535 {
		return fmt.Errorf("addr 端口必须在 1024 到 65535 之间")
	}
	return nil
}
