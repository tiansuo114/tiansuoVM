package i18n

import "context"

type ctxKey string

const ctxLanguageKey ctxKey = "language"

const (
	LangZH = "zh"
	LangEN = "en"

	AssetsInfoPrefix = "assetsInfo:"
)

func WithLanguage(ctx context.Context, lang string) context.Context {
	if ctx == nil {
		ctx = context.TODO()
	}

	return context.WithValue(ctx, ctxLanguageKey, lang)
}

func LanguageFromCtx(ctx context.Context) string {
	if ctx == nil {
		return ""
	}

	v := ctx.Value(ctxLanguageKey)
	if v != nil {
		return v.(string)
	}

	return ""
}
