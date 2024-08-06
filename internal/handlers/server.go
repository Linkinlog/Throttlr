package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"

	"github.com/linkinlog/throttlr/internal/db"
	"github.com/linkinlog/throttlr/internal/models"
)

var (
	InvalidAPIKey         = errors.New("invalid API key")
	InvalidEndpointValues = errors.New("invalid endpoint values")
	EndpointExists        = errors.New("endpoint already exists")
	EndpointMissing       = errors.New("endpoint doesnt exist")
	BucketExists          = errors.New("bucket already exists")
)

func apiLogHandler(l *slog.Logger, h HandlerErrorFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		httpErr := h(w, r)
		if httpErr != nil {
			l.Error("api handler error", "error", httpErr.Error())
			http.Error(w, httpErr.display, http.StatusInternalServerError)
		}
	}
}

func HandleServer(l *slog.Logger, ks *db.KeyStore, es *db.EndpointStore, bs *db.BucketStore) *http.ServeMux {
	m := http.NewServeMux()

	m.Handle("POST /register/{apiKey}", apiLogHandler(l, registerEndpoint(ks, es, bs)))
	m.Handle("/endpoints/{throttlrPath}", apiLogHandler(l, proxyEndpoint(ks, es)))

	return m
}

func proxyEndpoint(ks *db.KeyStore, es *db.EndpointStore) HandlerErrorFunc {
	return func(w http.ResponseWriter, r *http.Request) *httpError {
		key := r.URL.Query().Get("key")
		exists, apiKeyId := ks.Exists(key)
		if !exists {
			return &httpError{InvalidAPIKey, "No API key"}
		}
		if !ks.Valid(apiKeyId) {
			return &httpError{InvalidAPIKey, "Invalid API key"}
		}

		throttlrPath := r.PathValue("throttlrPath")

		e := &models.Endpoint{ThrottlrPath: throttlrPath}
		if exists, err := es.ExistsByThrottlr(r.Context(), e, apiKeyId); !exists {
			if err != nil {
				return &httpError{err, "failed to check if endpoint exists"}
			} else {
				return &httpError{EndpointMissing, "Endpoint doesnt exist"}
			}
		}

		if err := es.Fill(r.Context(), e, apiKeyId); err != nil {
			return &httpError{err, "failed to fill endpoint"}
		}

		url, err := url.Parse(e.OriginalUrl)
		if err != nil {
			return &httpError{err, "failed to parse original url"}
		}

		proxy := httputil.NewSingleHostReverseProxy(url)
		originalDirector := proxy.Director
		proxy.Director = func(req *http.Request) {
			originalDirector(req)
			modifyRequest(req, url)
		}

		proxy.ServeHTTP(w, r)
		return nil
	}
}

func modifyRequest(r *http.Request, originalUrl *url.URL) {
	r.URL = originalUrl
}

func registerEndpoint(ks *db.KeyStore, es *db.EndpointStore, bs *db.BucketStore) HandlerErrorFunc {
	return func(w http.ResponseWriter, r *http.Request) *httpError {
		key := r.PathValue("apiKey")
		exists, apiKeyId := ks.Exists(key)
		if !exists {
			return &httpError{InvalidAPIKey, "No API key"}
		}
		if !ks.Valid(apiKeyId) {
			return &httpError{InvalidAPIKey, "Invalid API key"}
		}
		if err := r.ParseForm(); err != nil {
			return &httpError{err, "failed to parse form"}
		}

		maxTokens, _ := strconv.Atoi(r.FormValue("max"))
		interval, _ := strconv.Atoi(r.FormValue("interval"))
		endpoint := r.FormValue("endpoint")

		if maxTokens == 0 || interval == 0 || endpoint == "" {
			return &httpError{InvalidEndpointValues, "Invalid endpoint values"}
		}

		e := models.NewEndpoint(key, endpoint)
		if exists, err := es.ExistsByOriginal(r.Context(), e, apiKeyId); exists {
			if err != nil {
				return &httpError{err, "failed to check if endpoint exists"}
			} else {
				return &httpError{EndpointExists, "Endpoint already exists"}
			}
		}

		apiKeyId, err := es.Store(r.Context(), e)
		if err != nil {
			return &httpError{err, "failed to store endpoint"}
		}

		b := models.NewBucket(e, models.Interval(interval), maxTokens)
		bDb := db.BucketModel{Bucket: b, EndpointId: apiKeyId}
		if exists, err := bs.Exists(r.Context(), bDb); exists {
			if err != nil {
				return &httpError{err, "failed to check if bucket exists"}
			} else {
				return &httpError{BucketExists, "Bucket already exists"}
			}
		}

		_, err = bs.Store(r.Context(), bDb)
		if err != nil {
			return &httpError{err, "failed to store bucket"}
		}

		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusCreated)
		_, err = w.Write([]byte(r.Host + "/endpoints/" + e.ThrottlrPath))
		if err != nil {
			return &httpError{err, "failed to write response"}
		}

		return nil
	}
}
