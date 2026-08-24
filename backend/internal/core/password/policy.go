package password

import "errors"

// Нууц үгийн дүрэм — ганц эх сурвалж (signup, нууц үг солих, гишүүн нэмэх,
// анхны админ бүгд үүнийг дуудна).
//
// Зөвхөн ASCII: латин үсэг, тоо, тусгай тэмдэгт. Кирилл болон бусад
// үсэг ХОРИОТОЙ — гар/бичгийн байрлал (layout) солигдоход хэрэглэгч өөрийн
// нууц үгээ дахин оруулж чадахгүй болдог, мөн шилжүүлэлт/хадгалалтын
// давхаргуудад (терминал, эх файл, гар утасны гар) төөрөгддөг.
const (
	MinLen = 8
	MaxLen = 128
)

var (
	ErrTooShort = errors.New("нууц үг 8+ тэмдэгт байх ёстой")
	ErrTooLong  = errors.New("нууц үг 128 тэмдэгтээс урт байж болохгүй")
	ErrCharset  = errors.New("нууц үгэнд зөвхөн латин үсэг (A-Z, a-z), тоо, тусгай тэмдэгт (!@#$%^&* гэх мэт) байна — кирилл болон зай хориотой")
	ErrWeak     = errors.New("нууц үгэнд латин үсэг, тоо, тусгай тэмдэгт гурвуулаа байх ёстой")
)

// Validate — дүрмийг шалгана. Алдааны мессежийг клиентэд шууд харуулж болно.
func Validate(p string) error {
	n := 0
	var hasLetter, hasDigit, hasSpecial bool
	for _, r := range p {
		n++
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z':
			hasLetter = true
		case r >= '0' && r <= '9':
			hasDigit = true
		case r > 0x20 && r < 0x7f: // ASCII-ийн бусад хэвлэгдэх тэмдэгт
			hasSpecial = true
		default:
			// Кирилл, зай, таб, эмодзи, хяналтын тэмдэгт бүгд энд унана.
			return ErrCharset
		}
	}
	switch {
	case n < MinLen:
		return ErrTooShort
	case n > MaxLen:
		return ErrTooLong
	case !hasLetter || !hasDigit || !hasSpecial:
		return ErrWeak
	}
	return nil
}
