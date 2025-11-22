package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// for now just a global variable
var urls map[string]string = make(map[string]string)

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/", handleMain)

	err := http.ListenAndServe("localhost:8080", mux)
	if err != nil {
		fmt.Printf("error: %v\n", err)
	}
}

// generate random url using rand package
func generateRandomUrl() string {
	randomUrl := make([]byte, 10) // for now buffer size is fixed size
	rand.Read(randomUrl) // Read function returns an error but it's always nil

	encodedUrl := hex.EncodeToString(randomUrl)
	return encodedUrl
}

// main handler. Both POST and GET requests
func handleMain(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		w.Header().Set("Content-Type", "text/plain")
		if r.URL.Path != "/" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		url, err := io.ReadAll(r.Body)
		if err != nil {
			fmt.Printf("error: %v\n", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if len(url) == 0 {
			fmt.Println("empty body")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		encodedUrl := generateRandomUrl()
		urls[encodedUrl] = string(url)

		fullUrl := "http://localhost:8080/"+encodedUrl
		
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(fullUrl))
		return
	case http.MethodGet:
		if r.URL.Path == "/" {
			fmt.Println("not a / route")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		url := r.URL.Path
		id := strings.TrimPrefix(url, "/")

		originalUrl, exists := urls[id]
		if !exists {
			fmt.Println("id doesn't exists in db")
			w.WriteHeader(http.StatusBadRequest)
			return
		} 

		http.Redirect(w, r, originalUrl, http.StatusTemporaryRedirect)
		return
	}

	w.WriteHeader(http.StatusMethodNotAllowed)
}