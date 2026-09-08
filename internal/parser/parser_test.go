package parser

import (
	"errors"
	"testing"
)

func TestFoundMatchesExactCardIdentifiers(t *testing.T) {
	body := []byte(`<html><body><div class="search-registry-entry-block"><span>ИНН: 6317024749</span><span>КПП: 631701001</span></div></body></html>`)
	found, err := Found(body, "6317024749", "631701001")
	if err != nil || !found {
		t.Fatalf("found=%v err=%v", found, err)
	}
}

func TestFoundDoesNotUseSearchFormValues(t *testing.T) {
	body := []byte(`<html><body><input name="inn" value="6317024749"><input name="kpp" value="631701001"><div>По вашему запросу ничего не найдено</div></body></html>`)
	found, err := Found(body, "6317024749", "631701001")
	if err != nil || found {
		t.Fatalf("found=%v err=%v", found, err)
	}
}

func TestFoundRejectsUnknownMarkup(t *testing.T) {
	_, err := Found([]byte(`<html><body>maintenance</body></html>`), "6317024749", "631701001")
	if !errors.Is(err, ErrUnexpectedMarkup) {
		t.Fatalf("err=%v", err)
	}
}
