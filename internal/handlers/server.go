package handlers

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"

	"github.com/linkinlog/throttlr/internal"
	"github.com/linkinlog/throttlr/internal/db"
	"github.com/linkinlog/throttlr/internal/models"
	"github.com/linkinlog/throttlr/web/partials"
)

var (
	MissingAPIKey         = errors.New("missing API key")
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
	m.Handle("GET /views/endpoints", apiLogHandler(l, handleEndpoints(es, ks)))

	return m
}

func handleEndpoints(es *db.EndpointStore, ks *db.KeyStore) HandlerErrorFunc {
	return func(w http.ResponseWriter, r *http.Request) *httpError {
		key := r.URL.Query().Get("apiKey")

		if key == "" {
			return &httpError{InvalidAPIKey, "No API key"}
		}

		exists, apiKeyId := ks.Exists(key)
		if !exists {
			return &httpError{InvalidAPIKey, "No API key"}
		}

		if !ks.Valid(apiKeyId) {
			return &httpError{InvalidAPIKey, "Invalid API key"}
		}

		userId, err := ks.UserIdFromKey(key)
		if err != nil {
			return &httpError{err, "failed to get user from key"}
		}

		endpoints, err := es.AllForUser(r.Context(), userId)
		if err != nil {
			return &httpError{err, "failed to get endpoints"}
		}

		callbackUrl := "http://localhost:8091"
		if url, err := internal.DefaultEnv.Get("SERVER_CALLBACK_URL"); err == nil {
			callbackUrl = url
		}

		err = partials.Endpoints(callbackUrl, key, endpoints).Render(r.Context(), w)
		if err != nil {
			return &httpError{err, "failed to render endpoints"}
		}

		return nil
	}
}

func proxyEndpoint(ks *db.KeyStore, es *db.EndpointStore) HandlerErrorFunc {
	return func(w http.ResponseWriter, r *http.Request) *httpError {
		key := r.URL.Query().Get("key")
		exists, apiKeyId := ks.Exists(key)
		if !exists {
			return &httpError{MissingAPIKey, "No API key"}
		}
		if !ks.Valid(apiKeyId) {
			return &httpError{InvalidAPIKey, "Invalid API key"}
		}

		userId, err := ks.UserIdFromKey(key)
		if err != nil {
			return &httpError{err, "failed to get user from key"}
		}

		throttlrPath := r.PathValue("throttlrPath")

		e := &models.Endpoint{ThrottlrPath: throttlrPath}
		if exists, err := es.ExistsByThrottlr(r.Context(), e, userId); !exists {
			if err != nil {
				return &httpError{err, "failed to check if endpoint exists"}
			} else {
				return &httpError{EndpointMissing, "Endpoint doesnt exist"}
			}
		}

		if err := es.Fill(r.Context(), e, userId); err != nil {
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
	r.Host = originalUrl.Host
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

		u, err := url.Parse(endpoint)
		if err != nil {
			return &httpError{err, "failed to parse endpoint"}
		}

		if u.Scheme == "" || u.Host == "" {
			return &httpError{InvalidEndpointValues, "Invalid endpoint values"}
		}

		if maxTokens == 0 || interval == 0 || endpoint == "" {
			return &httpError{InvalidEndpointValues, "Invalid endpoint values"}
		}

		b := models.NewBucket(models.Interval(interval), maxTokens)
		bDb := db.BucketModel{Bucket: b}
		if exists, err := bs.Exists(r.Context(), bDb); exists {
			if err != nil {
				return &httpError{err, "failed to check if bucket exists"}
			} else {
				return &httpError{BucketExists, "Bucket already exists"}
			}
		}

		bucketId, err := bs.Store(r.Context(), bDb)
		if err != nil {
			return &httpError{err, "failed to store bucket"}
		}

		userId, err := ks.UserIdFromKey(key)
		if err != nil {
			return &httpError{err, "failed to get user from key"}
		}

		e := models.NewEndpoint(endpoint)
		if exists, err := es.ExistsByOriginal(r.Context(), e, userId); exists {
			if err != nil {
				return &httpError{err, "failed to check if endpoint exists"}
			} else {
				return &httpError{EndpointExists, "Endpoint already exists"}
			}
		}

		_, err = es.Store(r.Context(), e, userId, bucketId)
		if err != nil {
			return &httpError{err, "failed to store endpoint"}
		}

		callbackUrl := "http://localhost:8091"
		if url, err := internal.DefaultEnv.Get("SERVER_CALLBACK_URL"); err == nil {
			callbackUrl = url
		}
		response := callbackUrl + "/endpoints/" + e.ThrottlrPath + "?key=" + key

		if r.Header.Get("Hx-Request") == "true" {
			response = fmt.Sprintf("Success! Endpoint registered at <a href='%s' target='_blank'>%s</a>", response, response)
		}

		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusCreated)

		_, err = w.Write([]byte(response))
		if err != nil {
			return &httpError{err, "failed to write response"}
		}

		return nil
	}
}
