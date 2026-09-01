package service

import (
	"strings"

	"github.com/n8node/aiapp/internal/model"
)

const uiLocaleKey = "ui.locale"

// Studio UI locales shipped upstream (including Russian).
var uiLocales = map[string]string{
	"en":    "English",
	"zh-CN": "简体中文",
	"ja":    "日本語",
	"ko":    "한국어",
	"es":    "Español",
	"pt-BR": "Português (Brasil)",
	"fr":    "Français",
	"de":    "Deutsch",
	"it":    "Italiano",
	"ru":    "Русский",
	"hi":    "हिन्दी",
	"ar":    "العربية",
}

func NormalizeUILocale(raw string) string {
	v := strings.TrimSpace(raw)
	if _, ok := uiLocales[v]; ok {
		return v
	}
	return ""
}

func DefaultUILocale() string { return "ru" }

func UILocaleCatalog() []map[string]string {
	order := []string{"ru", "en", "zh-CN", "ja", "ko", "es", "pt-BR", "fr", "de", "it", "hi", "ar"}
	out := make([]map[string]string, 0, len(order))
	for _, code := range order {
		out = append(out, map[string]string{"code": code, "label": uiLocales[code]})
	}
	return out
}

func ValidTrainingPurpose(v string) bool {
	switch v {
	case model.TrainingPurposeBehavior, model.TrainingPurposeFormat,
		model.TrainingPurposeExtraction, model.TrainingPurposeClassification:
		return true
	default:
		return false
	}
}
