package main

import "testing"

func TestDefaultAddressFromPort(t *testing.T) {
	address, err := defaultAddress("19444")
	if err != nil {
		t.Fatal(err)
	}
	if address != "127.0.0.1:19444" {
		t.Fatalf("地址错误: %s", address)
	}
	if _, err := defaultAddress("8080x"); err == nil {
		t.Fatal("应拒绝非纯端口")
	}
}

func TestRejectWildcardAddress(t *testing.T) {
	if err := validateAddress("0.0.0.0:19081"); err == nil {
		t.Fatal("应拒绝通配监听")
	}
	if err := validateAddress("127.0.0.1:19081"); err != nil {
		t.Fatal(err)
	}
}
