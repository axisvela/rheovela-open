package rheovela

import (
	"context"
	"testing"
)

func TestStaticTokenSource(t *testing.T) {
	tok, err := StaticTokenSource("abc").Token(context.Background())
	if err != nil || tok != "abc" {
		t.Fatalf("token=%q err=%v", tok, err)
	}
}

func TestTokenFuncSource(t *testing.T) {
	src := TokenFuncSource(func(context.Context) (string, error) { return "fn-tok", nil })
	tok, err := src.Token(context.Background())
	if err != nil || tok != "fn-tok" {
		t.Fatalf("token=%q err=%v", tok, err)
	}
}
