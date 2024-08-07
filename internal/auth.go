package internal

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gorilla/sessions"
	"github.com/jackc/pgx/v5"
	"github.com/linkinlog/throttlr/internal/db"
	"github.com/linkinlog/throttlr/internal/models"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/github"
	"github.com/markbates/goth/providers/google"
)

func SetupGothic(callbackUrl string) error {
	env := DefaultEnv

	ghKey, err := env.Get("GITHUB_KEY")
	if err != nil {
		return fmt.Errorf("failed to get GITHUB_KEY, %w", err)
	}

	ghSecret, err := env.Get("GITHUB_SECRET")
	if err != nil {
		return fmt.Errorf("failed to get GITHUB_SECRET, %w", err)
	}

	gk, err := env.Get("GOOGLE_KEY")
	if err != nil {
		return fmt.Errorf("failed to get GOOGLE_KEY, %w", err)
	}

	gs, err := env.Get("GOOGLE_SECRET")
	if err != nil {
		return fmt.Errorf("failed to get GOOGLE_SECRET, %w", err)
	}

	goth.UseProviders(
		github.New(ghKey, ghSecret, callbackUrl+"/auth/github/callback", "user:email"),
		google.New(gk, gs, callbackUrl+"/auth/google/callback", "https://www.googleapis.com/auth/userinfo.email", "https://www.googleapis.com/auth/userinfo.profile"),
	)

	return nil
}

func AuthenticateUserRequest(
	w http.ResponseWriter,
	r *http.Request,
	gs sessions.Store,
	us *db.UserStore,
) error {
	u, err := gothic.CompleteUserAuth(w, r)
	if err != nil {
		rErr := models.RemoveUserFromSession(r, w, gs)
		if rErr != nil {
			return fmt.Errorf("failed to remove user from session, %w, previous error: %w", rErr, err)
		}
		return err
	}
	uId := fmt.Sprintf("%s-%s", u.UserID, u.Provider)
	usr, err := us.ById(r.Context(), uId)

	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}

	if errors.Is(err, pgx.ErrNoRows) {
		usr = models.NewUser().
			SetId(uId).
			SetName(u.Name).
			SetEmail(u.Email)

		if err := us.Store(r.Context(), *usr); err != nil {
			return err
		}
	}

	sErr := usr.SaveToSession(r, w, gs)
	if sErr != nil {
		return sErr
	}
	return nil
}
