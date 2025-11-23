package handler

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// for now just a global variable
var Urls map[string]string = make(map[string]string)
var BaseURL string = "http://localhost:8080/"

// generate random url using rand package
func GenerateRandomUrl() string {
	randomUrl := make([]byte, 10) // for now buffer size is fixed size
	rand.Read(randomUrl)          // Read function returns an error but it's always nil

	encodedUrl := hex.EncodeToString(randomUrl)
	return encodedUrl
}

// handler POST URL
func HandlePost(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
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

		encodedUrl := GenerateRandomUrl()
		Urls[encodedUrl] = string(url)

		fullUrl := BaseURL + encodedUrl

		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(fullUrl))
	} else if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
	} else {
		w.WriteHeader(http.StatusBadRequest)
	}
}

// handler GET URL by ID
func HandleGetById(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		// if r.URL.Path == "/" {
		// 	fmt.Println("not a / route")
		// 	w.WriteHeader(http.StatusBadRequest)
		// 	return
		// }

		stringId := r.PathValue("id")
		stringId = strings.TrimSpace(stringId)
		fmt.Println(stringId)
		if stringId == "" {
			fmt.Println("empty id")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		// url := r.URL.Path
		// id := strings.TrimPrefix(url, "/")

		originalUrl, exists := Urls[stringId]
		fmt.Println(Urls)
		fmt.Println("original url", Urls[stringId])
		if !exists {
			fmt.Println("id doesn't exists in db")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		http.Redirect(w, r, originalUrl, http.StatusTemporaryRedirect)
	} else if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
	} else {
		w.WriteHeader(http.StatusBadRequest)
	}
}
