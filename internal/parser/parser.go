package parser

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

var ErrUnexpectedMarkup = errors.New("unexpected EIS search page markup")

var whitespace = regexp.MustCompile(`\s+`)

var noResultsMarkers = []string{
	"по вашему запросу ничего не найдено",
	"по вашему запросу записи не найдены",
	"по вашему запросу организаций не найдено",
	"всего записей: 0",
	"найдено: 0",
}

// Found проверяет только карточки результатов. Значения ИНН/КПП в форме поиска
// не учитываются, поэтому пустая выдача не превращается в ложный положительный ответ.
func Found(body []byte, inn, kpp string) (bool, error) {
	document, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return false, fmt.Errorf("parse HTML: %w", err)
	}

	cards := document.Find(".search-registry-entry-block")
	if cards.Length() == 0 {
		pageText := normalize(document.Text())
		for _, marker := range noResultsMarkers {
			if strings.Contains(pageText, marker) {
				return false, nil
			}
		}
		return false, ErrUnexpectedMarkup
	}

	found := false
	cards.EachWithBreak(func(_ int, card *goquery.Selection) bool {
		text := normalize(card.Text())
		if containsIdentifier(text, "инн", inn) && containsIdentifier(text, "кпп", kpp) {
			found = true
			return false
		}
		return true
	})
	return found, nil
}

func normalize(value string) string {
	return strings.ToLower(strings.TrimSpace(whitespace.ReplaceAllString(value, " ")))
}

func containsIdentifier(text, label, value string) bool {
	pattern := regexp.MustCompile(regexp.QuoteMeta(label) + `\s*:?\s*` + regexp.QuoteMeta(value) + `(?:[^0-9]|$)`)
	return pattern.MatchString(text)
}
