package handler

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPostURL_Success(t *testing.T) {
	originalURL := "https://youtube.com"

	body := strings.NewReader(originalURL)

	r := httptest.NewRequest("POST", BaseURL, body)
	w := httptest.NewRecorder()
	HandlePost(w, r)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusCreated {
		t.Errorf("incorrect status code. Got %v, wanted %v", res.StatusCode, http.StatusCreated)
	}

	if res.Header.Get("Content-Type") != "text/plain" {
		t.Errorf("incorrect Content-Type Header. Got %v, wanted %v", res.Header.Get("Content-Type"), "text/plain")
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		t.Errorf("error reading response body: %v", err)
	}

	shortURL := string(data)

	if !strings.HasPrefix(shortURL, BaseURL) {
		t.Errorf("expected short URL to start with %s, got %s", BaseURL, shortURL)
	}

	splitted := strings.Split(shortURL, "/")
	id := splitted[len(splitted)-1]

	if originalURL != Urls[id] {
		fmt.Println(shortURL)
		fmt.Println(originalURL, Urls[id])
		t.Errorf("failed to save shortened url.")
	}
}

func TestPostURL_EmptyURL(t *testing.T) {
	originalURL := ""
	body := strings.NewReader(originalURL)

	r := httptest.NewRequest("POST", BaseURL, body)
	w := httptest.NewRecorder()
	HandlePost(w, r)

	res := w.Result()
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("incorrect status code. Got %v, wanted %v", res.StatusCode, http.StatusBadRequest)
	}
}

func TestGetURL_Success(t *testing.T) {
	savedUrls := Urls
	testURL_DB := map[string]string{
		"e1ef4c662c790d8e4f72": "https://google.com",
	}

	Urls = testURL_DB
	defer func() {
		Urls = savedUrls
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("/{id}", HandleGetById)

	r := httptest.NewRequest("GET", BaseURL+"e1ef4c662c790d8e4f72", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusTemporaryRedirect {
		t.Errorf("incorrect status code. Got %v, wanted %v", res.StatusCode, http.StatusTemporaryRedirect)
	}

	location := res.Header.Get("Location")
	if location != "https://google.com" {
		t.Errorf("incorrect Location header. Got %v, wanted %v", location, "https://google.com")
	}
}

func TestGetURL_NonExistID(t *testing.T) {
	nonExistentID := "5f4e167e355b7b52571c"

	r := httptest.NewRequest("GET", BaseURL+nonExistentID, nil)
	w := httptest.NewRecorder()

	HandleGetById(w, r)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("incorrect status code. Got %v, wanted %v", res.StatusCode, http.StatusBadRequest)
	}
}

func TestGetURL_EmptyID(t *testing.T) {
	nonExistentID := ""

	r := httptest.NewRequest("GET", BaseURL+nonExistentID, nil)
	w := httptest.NewRecorder()

	HandleGetById(w, r)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("incorrect status code. Got %v, wanted %v", res.StatusCode, http.StatusBadRequest)
	}
}
