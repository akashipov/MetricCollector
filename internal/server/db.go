package server

import (
	"database/sql"
	"net/http"

	_ "github.com/lib/pq"
)

func TestConnectionPostgres(w http.ResponseWriter, request *http.Request) {
	db, err := sql.Open("postgres", *PsqlInfo)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer db.Close()
	err = db.Ping()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
