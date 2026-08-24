package password

import (
	"errors"
	"strings"
	"testing"
)

func TestValidate(t *testing.T) {
	good := []string{
		"password-12", "Abc123!x", "Xy9$Xy9$", "aaaaaaa1!", "P@ssw0rd",
		"1234567a!", strings.Repeat("a1!", 42) + "b", // 127 тэмдэгт
	}
	for _, p := range good {
		if err := Validate(p); err != nil {
			t.Errorf("зөв нууц үг татгалзав %q: %v", p, err)
		}
	}
	cases := map[string]struct {
		pass string
		want error
	}{
		"богино":          {"Ab1!", ErrTooShort},
		"хэт урт":         {strings.Repeat("a1!", 43) + "bb", ErrTooLong},
		"кирилл":          {"Нууцүг123!", ErrCharset},
		"кирилл нэг үсэг": {"passwоrd-12", ErrCharset}, // 'о' нь кирилл
		"зай":             {"pass word1!", ErrCharset},
		"эмодзи":          {"password1!🙂", ErrCharset},
		"таб":             {"password1!\t", ErrCharset},
		"зөвхөн үсэг":     {"passwordd", ErrWeak},
		"үсэг+тоо":        {"password12", ErrWeak},
		"тоо+тэмдэгт":     {"12345678!", ErrWeak},
		"үсэг+тэмдэгт":    {"password!!", ErrWeak},
		"хоосон":          {"", ErrTooShort},
	}
	for name, c := range cases {
		if err := Validate(c.pass); !errors.Is(err, c.want) {
			t.Errorf("%s (%q): алдаа = %v, хүлээсэн %v", name, c.pass, err, c.want)
		}
	}
	// Алдааны мессеж хэрэглэгчид ойлгомжтой (кирилл гэдгийг хэлнэ).
	if !strings.Contains(ErrCharset.Error(), "кирилл") || !strings.Contains(ErrWeak.Error(), "тусгай") {
		t.Fatal("алдааны мессеж тодорхойгүй")
	}
	// Батлагдсан нууц үг hash хийгдэж, буцаж шалгагдана.
	h, err := Hash("Abc123!x")
	if err != nil {
		t.Fatal(err)
	}
	if !Verify("Abc123!x", h) || Verify("Abc123!y", h) {
		t.Fatal("Hash/Verify")
	}
}
