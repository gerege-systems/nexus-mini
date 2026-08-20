package nexus

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/jackc/pgx/v5/pgconn"
)

// HTTP туслахууд — SDK-ийн хэсэг: модуль internal/-ээс юу ч импортлохгүй
// handler бичиж чадах ёстой (өмнө нь internal доторх httpx-д байсан нь
// гадны репогийн модульд компайл хийгдэхгүй байсан).

// JSON — хариу бичих.
func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// Error — алдааны хариу.
func Error(w http.ResponseWriter, status int, msg string) {
	JSON(w, status, map[string]string{"error": msg})
}

// Decode — биеийг задлана, 1MB хязгаартай, үл мэдэх талбар хориотой.
func Decode(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		Error(w, http.StatusBadRequest, "invalid request body")
		return false
	}
	return true
}

// IsUniqueViolation — Postgres 23505 (давхардсан түлхүүр) эсэх.
func IsUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// IsFKViolation — Postgres 23503 (байхгүй мөр рүү заасан) эсэх.
func IsFKViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}

// DBError — DB-ийн алдааг аюулгүй хариулна: давхардал бол 409 + өгсөн
// мессеж, бусад нь серверийн логт бүрэн, клиентэд ерөнхий 500 (raw pg
// алдааг клиентэд задалдаг байсныг аудит шүүмжилсэн).
func DBError(w http.ResponseWriter, err error, conflictMsg string) {
	if IsUniqueViolation(err) {
		Error(w, http.StatusConflict, conflictMsg)
		return
	}
	log.Printf("db error: %v", err)
	Error(w, http.StatusInternalServerError, "internal error")
}
