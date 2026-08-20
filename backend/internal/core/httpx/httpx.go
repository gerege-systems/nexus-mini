// Package httpx — pkg/nexus-ийн вэб туслахуудын дотоод alias.
// Хэрэгжүүлэлт нь SDK-д (pkg/nexus/web.go) амьдардаг: модулиуд internal-гүй
// компайл хийгдэх ёстой тул эх хувь нь тэнд.
package httpx

import (
	"net/http"

	"github.com/gerege-systems/nexus-mini/backend/pkg/nexus"
)

func JSON(w http.ResponseWriter, status int, v any) { nexus.JSON(w, status, v) }

func Error(w http.ResponseWriter, status int, msg string) { nexus.Error(w, status, msg) }

func Decode(w http.ResponseWriter, r *http.Request, v any) bool { return nexus.Decode(w, r, v) }

func DBError(w http.ResponseWriter, err error, conflictMsg string) {
	nexus.DBError(w, err, conflictMsg)
}
