package ai

import (
	"strings"
	"testing"
	"unicode"
)

func TestTokenizeChineseBigrams(t *testing.T) {
	tokens := tokenize("Pod CrashLoop 排查步骤")
	joined := strings.Join(tokens, ",")
	hasHanBigram := false
	for _, tok := range tokens {
		runes := []rune(tok)
		if len(runes) == 2 && unicode.Is(unicode.Han, runes[0]) {
			hasHanBigram = true
			break
		}
	}
	if !hasHanBigram {
		t.Fatalf("expected Chinese bigram in tokens: %s", joined)
	}
	found := false
	for _, tok := range tokens {
		if tok == "crashloop" || tok == "pod" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected latin tokens, got %s", joined)
	}
}
