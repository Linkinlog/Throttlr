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
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/linkinlog/throttlr/docs"
	"github.com/linkinlog/throttlr/internal"
	"github.com/linkinlog/throttlr/internal/db"
	"github.com/linkinlog/throttlr/internal/models"
	httpSwagger "github.com/swaggo/http-swagger/v2"
)

// @title						Throttlr API
// @version					0.0.1
// @description				This is the API for Throttlr, a rate limiting service.
// @BasePath					/v1
// @securityDefinitions.apikey	ApiKeyAuth
// @in							query
// @name						key
var (
	MissingAPIKey         = errors.New("missing API key")
	InvalidAPIKey         = errors.New("invalid API key")
	InvalidEndpointValues = errors.New("invalid endpoint values")
	EndpointExists        = errors.New("endpoint already exists")
	EndpointMissing       = errors.New("endpoint doesnt exist")
	BucketExists          = errors.New("bucket already exists")
)

func (e *endpoint) apiLogHandler(h HandlerErrorFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		e.l.Debug("hit", "method", r.Method, "path", r.URL.Path, "source", r.Header.Get("Cf-Connecting-IP"))
		httpErr := h(w, r)
		if httpErr != nil {
			status := http.StatusInternalServerError
			if errors.Is(httpErr.error, models.ErrBucketFull) {
				status = http.StatusTooManyRequests
			}
			http.Error(w, httpErr.display, status)
			e.l.Error("api handler error", "error", httpErr.Error())
		}
	}
}

func HandleServer(l *slog.Logger, pool *pgxpool.Pool) *http.ServeMux {
	m := http.NewServeMux()

	e := &endpoint{pool: pool, l: l}
	m.Handle("/v1/", http.StripPrefix("/v1", e.serveV1()))

	m.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("OK"))
		if err != nil {
			l.Error("health check failed", "error", err)
		}
	})

	m.Handle("GET /swagger/", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),                        // The url pointing to API definition
		httpSwagger.DefaultModelsExpandDepth(httpSwagger.HideModel), // Models will not be expanded
	))

	return m
}

type endpoint struct {
	pool *pgxpool.Pool
	l    *slog.Logger
}

func (e *endpoint) serveV1() *http.ServeMux {
	m := http.NewServeMux()

	m.Handle("POST /register", e.apiLogHandler(e.registerEndpoint()))
	m.Handle("POST /update/{throttlrPath}", e.apiLogHandler(e.updateEndpoint()))
	m.Handle("POST /delete/{throttlrPath}", e.apiLogHandler(e.deleteEndpoint()))
	m.Handle("/endpoints/{throttlrPath}", e.apiLogHandler(e.throttleEndpoint()))
	m.Handle("/proxy/{throttlrPath}", e.apiLogHandler(e.proxyEndpoint()))

	return m
}

// @Summary		Throttle endpoint
// @Description	Users will hit this endpoint to access the throttled endpoint
// @Tags			Throttlr
// @Accept			x-www-form-urlencoded
// @Accept			json
// @Produce		plain
// @Produce		json
// @Produce		html
// @Param			throttlrPath	path	string	true	"Throttlr path"
// @Security		ApiKeyAuth
// @Router			/endpoints/{throttlrPath} [get]
// @Router			/endpoints/{throttlrPath} [post]
// @Failure		429	{string}	string	"Too many requests"
func (e *endpoint) throttleEndpoint() HandlerErrorFunc {
	es := db.NewEndpointStore(e.pool, e.l)
	var m sync.Mutex
	return func(w http.ResponseWriter, r *http.Request) *httpError {
		if r.Context().Err() != nil {
			return &httpError{
				fmt.Errorf("throttle endpoint: %w", r.Context().Err()),
				"context error",
			}
		}
		m.Lock()
		defer m.Unlock()

		e, uId, httpErr := validateEndpointRequest(e.pool, r, e.l)
		if httpErr != nil {
			return httpErr
		}

		if e.Bucket == nil {
			return &httpError{
				fmt.Errorf("throttle endpoint: %w", models.ErrBucketNil),
				"Bucket is nil",
			}
		}

		dur := time.Minute
		switch e.Bucket.Interval {
		case models.Hour:
			dur = time.Hour
		case models.Day:
			dur = time.Hour * 24
		case models.Week:
			dur = time.Hour * 24 * 7
		case models.Month:
			dur = time.Hour * 24 * 30
		}

		if e.Bucket.WindowOpenedAt.Add(dur).Before(time.Now()) {
			e.Bucket.WindowOpenedAt = time.Now()
			e.Bucket.Current = 0
			if err := es.UpdateWindowOpenedAt(r.Context(), e, uId); err != nil {
				return &httpError{
					fmt.Errorf("throttle endpoint: failed to update window opened at: %w", err),
					"failed to update window opened at",
				}
			}
		}

		if e.Bucket.Current >= e.Bucket.Max {
			return &httpError{
				fmt.Errorf("throttle endpoint: %w", models.ErrBucketFull),
				"Rate limit reached, try again later or increase rate limit",
			}
		}

		proxy := httputil.NewSingleHostReverseProxy(e.OriginalUrl)
		originalDirector := proxy.Director
		proxy.Director = func(req *http.Request) {
			originalDirector(req)
			modifyRequest(req, e.OriginalUrl)
		}

		e.Bucket.Current = e.Bucket.Current + 1
		if err := es.UpdateBucketCount(r.Context(), e, uId); err != nil {
			return &httpError{
				fmt.Errorf("throttle endpoint: %w", err),
				"failed to update bucket",
			}
		}

		proxy.ServeHTTP(w, r)
		return nil
	}
}

// @Summary		Proxy endpoint
// @Description	Users will hit this endpoint to access the proxied endpoint
// @Tags			Proxy
// @Accept			x-www-form-urlencoded
// @Accept			json
// @Produce		plain
// @Produce		json
// @Produce		html
// @Param			throttlrPath	path	string	true	"Throttlr path"
// @Security		ApiKeyAuth
// @Router			/proxy/{throttlrPath} [get]
// @Router			/proxy/{throttlrPath} [post]
func (e *endpoint) proxyEndpoint() HandlerErrorFunc {
	return func(w http.ResponseWriter, r *http.Request) *httpError {
		e, _, httpErr := validateEndpointRequest(e.pool, r, e.l)
		if httpErr != nil {
			return httpErr
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
	// removes the key and any other query params
	r.URL = originalUrl
	// makes the destination think the request is coming from the proxy
	r.Host = originalUrl.Host
	// clears the cookies so we dont leak any session data
	r.Header.Del("Cookie")
}

// @Summary		Register endpoint
// @Description	Users will hit this endpoint to register a new endpoint
// @Tags			Register
// @Accept			x-www-form-urlencoded
// @Produce		plain
// @Produce		html
// @Security		ApiKeyAuth
// @Param			endpoint	formData	string	true	"Endpoint to register"
// @Param			interval	formData	int		true	"Interval, 1 = minute, 2 = hour, 3 = day, 4 = week, 5 = month"	Enums(1, 2, 3, 4, 5)
// @Param			max			formData	int		true	"Max requests per interval"
// @Success		201			{string}	string	"Created"
// @Router			/register [post]
func (e *endpoint) registerEndpoint() HandlerErrorFunc {
	ks := db.NewKeyStore(e.pool)
	es := db.NewEndpointStore(e.pool, e.l)

	return func(w http.ResponseWriter, r *http.Request) *httpError {
		endpoint, httpErr := validateRegisterEndpointRequest(r)
		if httpErr != nil {
			return httpErr
		}

		key, httpErr := validateApiKey(r, ks)
		if httpErr != nil {
			return httpErr
		}

		userId, err := ks.UserIdFromKey(key, r.Context())
		if err != nil {
			return &httpError{
				fmt.Errorf("register endpoint: failed to user id from key: %w", err),
				"failed to get user from key",
			}
		}

		e.l.Debug("register endpoint", "checking if exists", endpoint, "userId", userId)
		if exists, err := es.ExistsByOriginal(r.Context(), endpoint, userId); exists {
			e.l.Debug("register endpoint", "already exists", exists, "err", err)
			if err != nil {
				return &httpError{
					fmt.Errorf("register endpoint: failed to check if endpoint exists: %w", err),
					"failed to check if endpoint exists",
				}
			} else {
				return &httpError{
					fmt.Errorf("register endpoint: endpoint exists: %w", EndpointExists),
					"Endpoint already exists",
				}
			}
		}

		e.l.Debug("register endpoint", "storing", endpoint, "userId", userId)
		_, err = es.Store(r.Context(), endpoint, userId)
		if err != nil {
			e.l.Debug("register endpoint", "failed to store", err)
			return &httpError{
				fmt.Errorf("register endpoint: failed to store endpoint: %w", err),
				"failed to store endpoint",
			}
		}
		e.l.Debug("register endpoint", "stored", endpoint, "userId", userId)

		proxiedURL := fmt.Sprintf("%s/v1/endpoints/%s?key=%s",
			internal.ServerCallbackURL(),
			endpoint.ThrottlrPath,
			key,
		)

		if r.Header.Get("Hx-Request") == "true" {
			htmlResponse := fmt.Sprintf(
				"Success! Endpoint registered, try it <a href='%s' target='_blank'>here</a>",
				proxiedURL,
			)
			w.WriteHeader(http.StatusCreated)

			_, err = w.Write([]byte(htmlResponse))
			if err != nil {
				return &httpError{
					fmt.Errorf("register endpoint: failed to write response: %w", err),
					"failed to write response",
				}
			}
			return nil
		}

		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusCreated)

		_, err = w.Write([]byte(proxiedURL))
		if err != nil {
			return &httpError{
				fmt.Errorf("register endpoint: failed to write response: %w", err),
				"failed to write response",
			}
		}

		return nil
	}
}

// @Summary		Update endpoint
// @Description	Users will hit this endpoint to update an existing endpoint
// @Tags			Update
// @Accept			x-www-form-urlencoded
// @Produce		plain
// @Produce		html
// @Security		ApiKeyAuth
// @Param			endpoint		formData	string	true	"Updated endpoint"
// @Param			interval		formData	int		true	"Interval, 1 = minute, 2 = hour, 3 = day, 4 = week, 5 = month"	Enums(1, 2, 3, 4, 5)
// @Param			max				formData	int		true	"Max requests per interval"
// @Param			throttlrPath	path		string	true	"Throttlr path"
// @Success		201				{string}	string	"Created"
// @Router			/update/{throttlrPath} [post]
func (e *endpoint) updateEndpoint() HandlerErrorFunc {
	es := db.NewEndpointStore(e.pool, e.l)
	ks := db.NewKeyStore(e.pool)

	return func(w http.ResponseWriter, r *http.Request) *httpError {
		key, httpErr := validateApiKey(r, ks)
		if httpErr != nil {
			return httpErr
		}

		userId, err := ks.UserIdFromKey(key, r.Context())
		if err != nil {
			return &httpError{
				fmt.Errorf("update endpoint: failed to get user from key: %w", err),
				"failed to get user from key",
			}
		}

		if err := r.ParseForm(); err != nil {
			return &httpError{
				fmt.Errorf("update endpoint: failed to parse form: %w", err),
				"failed to parse json",
			}
		}

		newEndpoint := r.FormValue("endpoint")
		if newEndpoint == "" {
			return &httpError{
				fmt.Errorf("update endpoint: failed to parse new endpoint: %w", InvalidEndpointValues),
				"Invalid endpoint value",
			}
		}
		newInterval := r.FormValue("interval")
		newIntervalInt, err := strconv.Atoi(newInterval)
		if newInterval == "" || err != nil {
			return &httpError{
				fmt.Errorf("update endpoint: failed to parse new interval: %w", InvalidEndpointValues),
				"Invalid interval value",
			}
		}
		newMax := r.FormValue("max")
		newMaxInt, err := strconv.Atoi(newMax)
		if newMax == "" || err != nil {
			return &httpError{
				fmt.Errorf("update endpoint: failed to parse new max: %w", InvalidEndpointValues),
				"Invalid max value",
			}
		}

		throttlrPath := r.PathValue("throttlrPath")
		if throttlrPath == "" {
			return &httpError{
				fmt.Errorf("update endpoint: failed to parse throttlrPath: %w", InvalidEndpointValues),
				"Invalid throttlrPath value",
			}
		}

		endpoint, err := es.Get(r.Context(), throttlrPath, userId)
		if err != nil {
			return &httpError{
				fmt.Errorf("update endpoint: failed to get endpoint: %w", err),
				"failed to get endpoint",
			}
		}

		endpoint.Bucket.Max = newMaxInt
		endpoint.Bucket.Interval = models.Interval(newIntervalInt)
		endpoint.OriginalUrl, err = url.Parse(newEndpoint)
		if err != nil {
			return &httpError{
				fmt.Errorf("update endpoint: failed to parse new endpoint: %w", err),
				"failed to parse new endpoint",
			}
		}

		if err := es.Update(r.Context(), endpoint, userId); err != nil {
			if strings.Contains(err.Error(), "endpoint already exists") {
				return &httpError{
					fmt.Errorf("update endpoint: failed to update endpoint: %w", EndpointExists),
					"Endpoint already exists",
				}
			}
			return &httpError{
				fmt.Errorf("update endpoint: failed to update endpoint: %w", err),
				"failed to update endpoint",
			}
		}

		proxiedURL := fmt.Sprintf("%s/v1/endpoints/%s?key=%s",
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
				return &httpError{
					fmt.Errorf("update endpoint: failed to write response: %w", err),
					"failed to write response",
				}
			}
			return nil
		}

		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusCreated)

		_, err = w.Write([]byte(proxiedURL))
		if err != nil {
			return &httpError{
				fmt.Errorf("update endpoint: failed to write response: %w", err),
				"failed to write response",
			}
		}

		return nil
	}
}

// @Summary		Delete endpoint
// @Description	Users will hit this endpoint to delete an existing endpoint
// @Tags			Delete
// @Accept			x-www-form-urlencoded
// @Produce		plain
// @Produce		html
// @Security		ApiKeyAuth
// @Param			throttlrPath	path		string	true	"Throttlr path"
// @Success		200				{string}	string	"Deleted"
// @Router			/delete/{throttlrPath} [post]
func (e *endpoint) deleteEndpoint() HandlerErrorFunc {
	ks := db.NewKeyStore(e.pool)
	es := db.NewEndpointStore(e.pool, e.l)

	return func(w http.ResponseWriter, r *http.Request) *httpError {
		key, httpErr := validateApiKey(r, ks)
		if httpErr != nil {
			return httpErr
		}

		userId, err := ks.UserIdFromKey(key, r.Context())
		if err != nil {
			return &httpError{
				fmt.Errorf("delete endpoint: failed to get user from key: %w", err),
				"failed to get user from key",
			}
		}

		throttlrPath := r.PathValue("throttlrPath")
		if throttlrPath == "" {
			return &httpError{
				fmt.Errorf("delete endpoint: failed to parse throttlrPath: %w", InvalidEndpointValues),
				"Invalid throttlrPath value",
			}
		}

		e, err := es.Get(r.Context(), throttlrPath, userId)
		if err != nil {
			return &httpError{
				fmt.Errorf("delete endpoint: failed to get endpoint: %w", err),
				"failed to get endpoint",
			}
		}

		if err := es.Delete(r.Context(), e, userId); err != nil {
			return &httpError{
				fmt.Errorf("delete endpoint: failed to delete endpoint: %w", err),
				"failed to delete endpoint",
			}
		}

		url := internal.ClientCallbackURL()
		w.Header().Set("location", url+"/")
		w.WriteHeader(http.StatusSeeOther)

		return nil
	}
}

func validateEndpointRequest(pool *pgxpool.Pool, r *http.Request, l *slog.Logger) (*models.Endpoint, string, *httpError) {
	ks := db.NewKeyStore(pool)
	es := db.NewEndpointStore(pool, l)

	key := r.URL.Query().Get("key")
	exists, apiKeyId := ks.Exists(key, r.Context())
	if !exists {
		return nil, "", &httpError{
			fmt.Errorf("proxy error: %w, key: %s, apiKeyId: %d", MissingAPIKey, key, apiKeyId),
			"No API key",
		}
	}
	if !ks.Valid(apiKeyId, r.Context()) {
		return nil, "", &httpError{
			fmt.Errorf("proxy error: %w, key: %s, apiKeyId: %d", InvalidAPIKey, key, apiKeyId),
			"Invalid API key",
		}
	}

	userId, err := ks.UserIdFromKey(key, r.Context())
	if err != nil {
		return nil, "", &httpError{
			fmt.Errorf("proxy error: %w, key: %s, apiKeyId: %d",
				err,
				key,
				apiKeyId,
			),
			"failed to get user from key",
		}
	}

	throttlrPath := r.PathValue("throttlrPath")

	e := &models.Endpoint{ThrottlrPath: throttlrPath, Bucket: &models.Bucket{}}
	if exists, err := es.ExistsByThrottlr(r.Context(), e, userId); !exists {
		if err != nil {
			return nil, "", &httpError{
				fmt.Errorf("proxy error: %w, key: %s, apiKeyId: %d, throttlrPath: %s, userId: %s",
					err,
					key,
					apiKeyId,
					throttlrPath,
					userId,
				),
				"failed to check if endpoint exists",
			}
		} else {
			return nil, "", &httpError{
				fmt.Errorf("proxy error: %w, key: %s, apiKeyId: %d, throttlrPath: %s, userId: %s",
					EndpointMissing,
					key,
					apiKeyId,
					throttlrPath,
					userId,
				),
				"Endpoint doesnt exist",
			}
		}
	}

	if err := es.Fill(r.Context(), e, userId); err != nil {
		return nil, "", &httpError{
			fmt.Errorf("proxy error: %w, key: %s, apiKeyId: %d, throttlrPath: %s, userId: %s",
				err,
				key,
				apiKeyId,
				throttlrPath,
				userId,
			),
			"failed to fill endpoint",
		}
	}

	return e, userId, nil
}

func validateRegisterEndpointRequest(r *http.Request) (*models.Endpoint, *httpError) {
	if err := r.ParseForm(); err != nil {
		return nil, &httpError{
			fmt.Errorf("validateEndpointRequest: failed to parse form: %w", err),
			"failed to parse form",
		}
	}

	maxTokens, _ := strconv.Atoi(r.FormValue("max"))
	interval, _ := strconv.Atoi(r.FormValue("interval"))
	endpoint := r.FormValue("endpoint")

	if maxTokens == 0 || interval == 0 || endpoint == "" {
		return nil, &httpError{
			fmt.Errorf("validateEndpointRequest: %w", InvalidEndpointValues),
			"Invalid endpoint values",
		}
	}
	b := models.NewBucket(models.Interval(interval), maxTokens)
	e, err := models.NewEndpoint(endpoint, b)
	if err != nil {
		return nil, &httpError{
			fmt.Errorf("validateEndpointRequest: failed to create endpoint: %w", err),
			"failed to create endpoint",
		}
	}

	return e, nil
}

func validateApiKey(r *http.Request, ks *db.KeyStore) (string, *httpError) {
	key := r.URL.Query().Get("key")
	exists, apiKeyId := ks.Exists(key, r.Context())
	if !exists {
		return "", &httpError{
			fmt.Errorf("validateApiKey: failed to validate key: %w", MissingAPIKey),
			"No API key",
		}
	}
	if !ks.Valid(apiKeyId, r.Context()) {
		return "", &httpError{
			fmt.Errorf("validateApiKey: failed to validate key: %w", InvalidAPIKey),
			"Invalid API key",
		}
	}
	return key, nil
}
