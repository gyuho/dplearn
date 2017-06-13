package urlutil

import (
	"fmt"
	"testing"

	humanize "github.com/dustin/go-humanize"
)

func TestTrimQuery(t *testing.T) {
	ep := "https://images.pexels.com/photos/127028/pexels-photo-127028.jpeg?w=1260&h=750&auto=compress&cs=tinysrgb"
	exp := "https://images.pexels.com/photos/127028/pexels-photo-127028.jpeg"
	if TrimQuery(ep) != exp {
		t.Fatalf("expected %q, got %q", exp, TrimQuery(ep))
	}
}

func TestGetContentLength(t *testing.T) {
	ep := "https://images.pexels.com/photos/127028/pexels-photo-127028.jpeg?w=1260&h=750&auto=compress&cs=tinysrgb"
	size, sizet, err := GetContentLength(TrimQuery(ep))
	if err != nil {
		t.Skip(err)
	}
	fmt.Println(size, sizet)
}

func TestGet(t *testing.T) {
	ep := "https://images.pexels.com/photos/127028/pexels-photo-127028.jpeg?w=1260&h=750&auto=compress&cs=tinysrgb"
	data, err := Get(TrimQuery(ep))
	if err != nil {
		t.Skip(err)
	}
	fmt.Println(humanize.Bytes(uint64(len(data))))
}
