package main

import "net/http"

func main() {
	http.HandleFunc("/", handle)

	if err := http.ListenAndServe(":80", nil); err != nil {
		panic(err)
	}
}

func handle(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("hello"))
}
