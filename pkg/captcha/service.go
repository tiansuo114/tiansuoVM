package captcha

import (
	"github.com/mojocn/base64Captcha"
	"image/color"
	"strings"
	"time"
)

type store struct {
	base64Captcha.Store
}

func (s *store) Verify(id, answer string, clear bool) bool {
	v := s.Get(id, clear)
	return strings.ToLower(v) == strings.ToLower(answer)
}

var (
	defaultDriver base64Captcha.Driver = base64Captcha.NewDriverMath(
		100, 200, 0,
		0,
		&color.RGBA{
			R: 255,
			G: 255,
			B: 255,
			A: 255,
		}, nil, nil)
	// defaultDriver base64Captcha.Driver = base64Captcha.NewDriverString(
	// 	100, 200, 3,
	// 	base64Captcha.OptionShowHollowLine,
	// 	4,
	// 	base64Captcha.TxtAlphabet+base64Captcha.TxtNumbers,
	// 	&color.RGBA{
	// 		R: 40,
	// 		G: 30,
	// 		B: 89,
	// 		A: 29,
	// 	}, nil, nil)

	defaultStore = &store{Store: base64Captcha.NewMemoryStore(base64Captcha.GCLimitNumber, 1*time.Minute)}
)

func ReplaceDriver(driver base64Captcha.Driver) {
	defaultDriver = driver
}

func CreateCaptcha() (string, string, error) {
	c := base64Captcha.NewCaptcha(defaultDriver, defaultStore)
	id, b64s, err := c.Generate()
	return id, b64s, err
}

func VerifyCaptcha(id, VerifyValue string) bool {
	return defaultStore.Verify(id, VerifyValue, true)
}

func PeekCaptchaValue(id string) string {
	return defaultStore.Get(id, false)
}
