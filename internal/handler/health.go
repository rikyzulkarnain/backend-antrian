package handler

import "net/http"

func Health(w http.ResponseWriter, _ *http.Request) {
	ok(w, map[string]string{"status": "ok"})
}
