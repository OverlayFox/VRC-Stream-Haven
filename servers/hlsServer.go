package servers

import "net/http"

func addHeaders(h http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		h.ServeHTTP(w, r)
	}
}

func ServeM3u8() {
	http.Handle("/", addHeaders(http.FileServer(http.Dir("streams"))))

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		return
	}
}
