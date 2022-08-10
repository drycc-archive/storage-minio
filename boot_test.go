package main

import (
	"os"
	"testing"
)

func TestNewMinioClient(t *testing.T) {
	os.Args = []string{"boot", "unknow-cmd", "server", "http://node{1...16}.example.com/mnt/export{1...32}"}
	main()
	if len(os.Args) != 3 {
		t.Fatalf("unexpected args len")
	}
}
