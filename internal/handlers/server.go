package handlers

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/linkinlog/throttlr/internal"
	"github.com/linkinlog/throttlr/internal/db"
	"github.com/linkinlog/throttlr/internal/models"
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

func HandleServer(l *slog.Logger, pool *pgx.Conn) *http.ServeMux {
	m := http.NewServeMux()

	m.Handle("POST /register/{apiKey}", apiLogHandler(l, registerEndpoint(pool)))
	m.Handle("POST /update/{apiKey}", apiLogHandler(l, updateEndpoint(pool)))
	m.Handle("POST /delete/{apiKey}", apiLogHandler(l, deleteEndpoint(pool)))
	m.Handle("/endpoints/{throttlrPath}", apiLogHandler(l, proxyEndpoint(pool)))

	return m
}

func proxyEndpoint(pool *pgx.Conn) HandlerErrorFunc {
	ks := db.NewKeyStore(pool)
	es := db.NewEndpointStore(pool)

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

		e := &models.Endpoint{ThrottlrPath: throttlrPath, Bucket: &models.Bucket{}}
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

		proxy := httputil.NewSingleHostReverseProxy(e.OriginalUrl)
		originalDirector := proxy.Director
		proxy.Director = func(req *http.Request) {
			originalDirector(req)
			modifyRequest(req, e.OriginalUrl)
		}

		proxy.ServeHTTP(w, r)
		return nil
	}
}

func modifyRequest(r *http.Request, originalUrl *url.URL) {
	r.URL = originalUrl
	r.Host = originalUrl.Host
}

func registerEndpoint(pool *pgx.Conn) HandlerErrorFunc {
	ks := db.NewKeyStore(pool)
	es := db.NewEndpointStore(pool)

	return func(w http.ResponseWriter, r *http.Request) *httpError {
		endpoint, key, httpErr := validateEndpointRequest(r, ks)
		if httpErr != nil {
			return httpErr
		}

		userId, err := ks.UserIdFromKey(key)
		if err != nil {
			return &httpError{err, "failed to get user from key"}
		}

		if exists, err := es.ExistsByOriginal(r.Context(), endpoint, userId); exists {
			if err != nil {
				return &httpError{err, "failed to check if endpoint exists"}
			} else {
				return &httpError{EndpointExists, "Endpoint already exists"}
			}
		}

		_, err = es.Store(r.Context(), endpoint, userId)
		if err != nil {
			return &httpError{err, "failed to store endpoint"}
		}

		proxiedURL := fmt.Sprintf("%s/endpoints/%s?key=%s",
			internal.ServerCallbackURL(),
			endpoint.ThrottlrPath,
			key,
		)

		if r.Header.Get("Hx-Request") == "true" {
			htmlResponse := fmt.Sprintf(
				"Success! Endpoint registered at <a href='%s' target='_blank'>Here</a>",
				proxiedURL,
			)
			w.WriteHeader(http.StatusCreated)

			_, err = w.Write([]byte(htmlResponse))
			if err != nil {
				return &httpError{err, "failed to write response"}
			}
			return nil
		}

		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusCreated)

		_, err = w.Write([]byte(proxiedURL))
		if err != nil {
			return &httpError{err, "failed to write response"}
		}

		return nil
	}
}

func updateEndpoint(pool *pgx.Conn) HandlerErrorFunc {
	ks := db.NewKeyStore(pool)
	es := db.NewEndpointStore(pool)

	return func(w http.ResponseWriter, r *http.Request) *httpError {
		key, httpErr := validateApiKey(r, ks)
		if httpErr != nil {
			return httpErr
		}

		userId, err := ks.UserIdFromKey(key)
		if err != nil {
			return &httpError{err, "failed to get user from key"}
		}

		if err := r.ParseForm(); err != nil {
			return &httpError{err, "failed to parse form"}
		}

		newEndpoint := r.FormValue("endpoint")
		if newEndpoint == "" {
			return &httpError{InvalidEndpointValues, "Invalid endpoint values"}
		}
		newInterval := r.FormValue("interval")
		newIntervalInt, err := strconv.Atoi(newInterval)
		if newInterval == "" || err != nil {
			return &httpError{InvalidEndpointValues, "Invalid endpoint values"}
		}
		newMax := r.FormValue("max")
		newMaxInt, err := strconv.Atoi(newMax)
		if newMax == "" || err != nil {
			return &httpError{InvalidEndpointValues, "Invalid endpoint values"}
		}
		endpointId := r.FormValue("endpoint_id")
		if endpointId == "" {
			return &httpError{InvalidEndpointValues, "Invalid endpoint values"}
		}

		enpointIdInt, err := strconv.Atoi(endpointId)
		if err != nil {
			return &httpError{err, "failed to convert endpoint id"}
		}

		endpoint, err := es.Get(r.Context(), enpointIdInt, userId)
		if err != nil {
			return &httpError{err, "failed to get endpoint"}
		}

		endpoint.Id = enpointIdInt
		endpoint.Bucket.Max = newMaxInt
		endpoint.Bucket.Interval = models.Interval(newIntervalInt)
		endpoint.OriginalUrl, err = url.Parse(newEndpoint)
		if err != nil {
			return &httpError{err, "failed to parse new endpoint"}
		}

		if err := es.Update(r.Context(), endpoint, userId); err != nil {
			if strings.Contains(err.Error(), "endpoint already exists") {
				return &httpError{EndpointExists, "Endpoint already exists"}
			}
			return &httpError{err, "failed to update endpoint"}
		}

		proxiedURL := fmt.Sprintf("%s/endpoints/%s?key=%s",
			internal.ServerCallbackURL(),
			endpoint.ThrottlrPath,
			key,
		)

		if r.Header.Get("Hx-Request") == "true" {
			htmlResponse := fmt.Sprintf(
				"Success! Endpoint updated <a href='%s' target='_blank'>Here</a>",
				proxiedURL,
			)
			w.WriteHeader(http.StatusCreated)

			_, err = w.Write([]byte(htmlResponse))
			if err != nil {
				return &httpError{err, "failed to write response"}
			}
			return nil
		}

		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusCreated)

		_, err = w.Write([]byte(proxiedURL))
		if err != nil {
			return &httpError{err, "failed to write response"}
		}

		return nil
	}
}

func deleteEndpoint(pool *pgx.Conn) HandlerErrorFunc {
	ks := db.NewKeyStore(pool)
	es := db.NewEndpointStore(pool)

	return func(w http.ResponseWriter, r *http.Request) *httpError {
		key, httpErr := validateApiKey(r, ks)
		if httpErr != nil {
			return httpErr
		}

		userId, err := ks.UserIdFromKey(key)
		if err != nil {
			return &httpError{err, "failed to get user from key"}
		}

		endpointId := r.URL.Query().Get("id")
		if endpointId == "" {
			return &httpError{InvalidEndpointValues, "Invalid endpoint values"}
		}

		enpointIdInt, err := strconv.Atoi(endpointId)
		if err != nil {
			return &httpError{err, "failed to convert endpoint id"}
		}

		endpoint, err := es.Get(r.Context(), enpointIdInt, userId)
		if err != nil {
			return &httpError{err, "failed to get endpoint"}
		}

		endpoint.Id = enpointIdInt

		if err := es.Delete(r.Context(), endpoint, userId); err != nil {
			return &httpError{err, "failed to delete endpoint"}
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)
		return nil
	}
}

func validateEndpointRequest(r *http.Request, ks *db.KeyStore) (*models.Endpoint, string, *httpError) {
	key, httpErr := validateApiKey(r, ks)
	if httpErr != nil {
		return nil, "", httpErr
	}

	if err := r.ParseForm(); err != nil {
		return nil, "", &httpError{err, "failed to parse form"}
	}

	maxTokens, _ := strconv.Atoi(r.FormValue("max"))
	interval, _ := strconv.Atoi(r.FormValue("interval"))
	endpoint := r.FormValue("endpoint")

	if maxTokens == 0 || interval == 0 || endpoint == "" {
		return nil, "", &httpError{InvalidEndpointValues, "Invalid endpoint values"}
	}
	b := models.NewBucket(models.Interval(interval), maxTokens)
	e, err := models.NewEndpoint(endpoint, b)
	if err != nil {
		return nil, "", &httpError{err, "failed to create endpoint"}
	}

	return e, key, nil
}

func validateApiKey(r *http.Request, ks *db.KeyStore) (string, *httpError) {
	key := r.PathValue("apiKey")
	exists, apiKeyId := ks.Exists(key)
	if !exists {
		return "", &httpError{InvalidAPIKey, "No API key"}
	}
	if !ks.Valid(apiKeyId) {
		return "", &httpError{InvalidAPIKey, "Invalid API key"}
	}
	return key, nil
}
