package auth

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"sort"

	"github.com/go-chi/chi/v5"
	"github.com/gochi-demo/internal/config"
	"github.com/gorilla/sessions"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/google"
)

const (
	key    = "somerandomkey12345678901234567890" // Must be 32 bytes for AES-256
	maxAge = 86400 * 30
	isProd = false
)

type ProviderIndex struct {
	Providers    []string
	ProvidersMap map[string]string
}

var indexTemplate = `{{range $key,$value:=.Providers}}
    <p><a href="/auth/{{$value}}">Log in with {{index $.ProvidersMap $value}}</a></p>
{{end}}`

var userTemplate = `
<p><a href="/logout/{{.Provider}}">logout</a></p>
<p>Name: {{.Name}} [{{.LastName}}, {{.FirstName}}]</p>
<p>Email: {{.Email}}</p>
<p>NickName: {{.NickName}}</p>
<p>Location: {{.Location}}</p>
<p>AvatarURL: {{.AvatarURL}} <img src="{{.AvatarURL}}"></p>
<p>Description: {{.Description}}</p>
<p>UserID: {{.UserID}}</p>
<p>AccessToken: {{.AccessToken}}</p>
<p>ExpiresAt: {{.ExpiresAt}}</p>
<p>RefreshToken: {{.RefreshToken}}</p>
`

// GetProvider is a middleware that extracts the provider from Chi URL params
// and adds it to the request context for Gothic to use
func GetProvider(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		provider := chi.URLParam(r, "provider")
		if provider == "" {
			http.Error(w, "Provider not found", http.StatusBadRequest)
			return
		}

		// Add provider to query params for Gothic
		q := r.URL.Query()
		q.Set("provider", provider)
		r.URL.RawQuery = q.Encode()

		// Also add to context
		ctx := context.WithValue(r.Context(), "provider", provider)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func NewAuth(r *chi.Mux) {
	googleClientId := config.GetConfig("CLIENT_ID")
	googleClientSecret := config.GetConfig("CLIENT_SECRET")
	googleClientCallbackURL := config.GetConfig("CLIENT_CALLBACK_URL")

	// Configure session store with proper settings
	sessionStore := sessions.NewCookieStore([]byte(key))
	sessionStore.MaxAge(maxAge)
	sessionStore.Options.Path = "/"
	sessionStore.Options.HttpOnly = true
	sessionStore.Options.Secure = isProd
	sessionStore.Options.SameSite = http.SameSiteLaxMode // Important for OAuth flow

	gothic.Store = sessionStore

	goth.UseProviders(
		google.New(googleClientId, googleClientSecret, googleClientCallbackURL, "email", "profile"),
	)

	m := map[string]string{
		"google": "Google",
	}

	var keys []string
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	providerIndex := &ProviderIndex{Providers: keys, ProvidersMap: m}

	// Auth start - wrap with GetProvider middleware
	r.With(GetProvider).Get("/auth/{provider}", func(res http.ResponseWriter, req *http.Request) {
		// Don't try CompleteUserAuth here on initial auth - it will fail
		gothic.BeginAuthHandler(res, req)
	})

	// Callback handler - wrap with GetProvider middleware
	r.With(GetProvider).Get("/auth/{provider}/callback", func(res http.ResponseWriter, req *http.Request) {
		user, err := gothic.CompleteUserAuth(res, req)
		if err != nil {
			fmt.Fprintf(res, "CompleteUserAuth failed: %v\n", err)
			return
		}

		fmt.Println("User authenticated:", user.Email)

		// Render user template or redirect
		t, _ := template.New("foo").Parse(userTemplate)
		t.Execute(res, user)
	})

	// Logout handler - wrap with GetProvider middleware
	r.With(GetProvider).Get("/logout/{provider}", func(res http.ResponseWriter, req *http.Request) {
		err := gothic.Logout(res, req)
		if err != nil {
			fmt.Fprintf(res, "Logout failed: %v\n", err)
			return
		}
		http.Redirect(res, req, "/", http.StatusTemporaryRedirect)
	})

	r.Get("/withauth", func(res http.ResponseWriter, req *http.Request) {
		t, _ := template.New("foo").Parse(indexTemplate)
		t.Execute(res, providerIndex)
	})
}
