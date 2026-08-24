package handlers

// grantError — errors.As-аар барих, мессежээ хадгалах гэрээ (цагаан хайрцаг).

import (
	"errors"
	"fmt"
	"testing"
)

func TestGrantErrorContract(t *testing.T) {
	var err error = grantError{msg: "өөрт байхгүй эрх: core.audit.read"}
	if err.Error() != "өөрт байхгүй эрх: core.audit.read" {
		t.Fatalf("Error() = %q", err.Error())
	}
	// Боож (wrap) хийсэн ч errors.As барина — handler-ууд ингэж 400 буцаадаг.
	wrapped := fmt.Errorf("гүйлгээ: %w", err)
	var ge grantError
	if !errors.As(wrapped, &ge) || ge.msg != "өөрт байхгүй эрх: core.audit.read" {
		t.Fatalf("errors.As = %v / %+v", errors.As(wrapped, &ge), ge)
	}
	// Өөр алдаатай хольж болохгүй.
	if errors.As(errors.New("энгийн"), &ge) {
		t.Fatal("энгийн алдааг grantError гэж таньлаа")
	}
	// %v форматлахад мессеж гарна (лог-д хэрэгтэй).
	if got := fmt.Sprintf("%v", err); got != "өөрт байхгүй эрх: core.audit.read" {
		t.Fatalf("%%v = %q", got)
	}
}
