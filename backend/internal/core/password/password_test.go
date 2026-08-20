package password

import "testing"

func TestHashVerify(t *testing.T) {
	h, err := Hash("нууц-үг-1234")
	if err != nil {
		t.Fatal(err)
	}
	if !Verify("нууц-үг-1234", h) {
		t.Fatal("зөв нууц үг танигдсангүй")
	}
	if Verify("буруу", h) {
		t.Fatal("буруу нууц үг танигдав")
	}
	if Verify("нууц-үг-1234", "$argon2id$гэмтсэн") {
		t.Fatal("гэмтсэн hash дээр true буцаав")
	}
}
