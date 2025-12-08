package auth

import (
	"context"
	"encoding/json"
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
	key            = "somerandomkey12345678901234567890" // Must be 32 bytes for AES-256
	maxAge         = 86400 * 30
	isProd         = false
	SessionName    = "user-session"
	UserSessionKey = "user"
)

// User represents the authenticated user data we store in session
type User struct {
	ID           string
	Email        string
	Name         string
	FirstName    string
	LastName     string
	AvatarURL    string
	Provider     string
	AccessToken  string
	RefreshToken string
}

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

var store *sessions.CookieStore

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

// RequireAuth is a middleware that protects routes requiring authentication
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, err := GetUserFromSession(r)
		if err != nil || user == nil {
			// Redirect to login page
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// Add user to context for the next handler
		ctx := context.WithValue(r.Context(), UserSessionKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// SaveUserToSession saves the user data to the session
func SaveUserToSession(w http.ResponseWriter, r *http.Request, gothUser goth.User) error {
	session, err := store.Get(r, SessionName)
	if err != nil {
		return err
	}

	user := &User{
		ID:           gothUser.UserID,
		Email:        gothUser.Email,
		Name:         gothUser.Name,
		FirstName:    gothUser.FirstName,
		LastName:     gothUser.LastName,
		AvatarURL:    gothUser.AvatarURL,
		Provider:     gothUser.Provider,
		AccessToken:  gothUser.AccessToken,
		RefreshToken: gothUser.RefreshToken,
	}

	// Serialize user to JSON
	userData, err := json.Marshal(user)
	if err != nil {
		return err
	}

	session.Values[UserSessionKey] = string(userData)
	return session.Save(r, w)
}

// GetUserFromSession retrieves the user data from the session
func GetUserFromSession(r *http.Request) (*User, error) {
	session, err := store.Get(r, SessionName)
	if err != nil {
		return nil, err
	}

	userDataStr, ok := session.Values[UserSessionKey].(string)
	if !ok || userDataStr == "" {
		return nil, fmt.Errorf("no user in session")
	}

	var user User
	err = json.Unmarshal([]byte(userDataStr), &user)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

// GetUserFromContext retrieves the user from the request context (set by RequireAuth middleware)
func GetUserFromContext(r *http.Request) (*User, error) {
	user, ok := r.Context().Value(UserSessionKey).(*User)
	if !ok || user == nil {
		return nil, fmt.Errorf("no user in context")
	}
	return user, nil
}

// ClearUserSession removes the user from the session
func ClearUserSession(w http.ResponseWriter, r *http.Request) error {
	session, err := store.Get(r, SessionName)
	if err != nil {
		return err
	}

	delete(session.Values, UserSessionKey)
	return session.Save(r, w)
}

func NewAuth(r *chi.Mux) {
	googleClientId := config.GetConfig("CLIENT_ID")
	googleClientSecret := config.GetConfig("CLIENT_SECRET")
	googleClientCallbackURL := config.GetConfig("CLIENT_CALLBACK_URL")

	// Configure session store with proper settings
	store = sessions.NewCookieStore([]byte(key))
	store.MaxAge(maxAge)
	store.Options.Path = "/"
	store.Options.HttpOnly = true
	store.Options.Secure = isProd
	store.Options.SameSite = http.SameSiteLaxMode

	gothic.Store = store

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
		gothic.BeginAuthHandler(res, req)
	})

	// Callback handler - wrap with GetProvider middleware
	r.With(GetProvider).Get("/auth/{provider}/callback", func(res http.ResponseWriter, req *http.Request) {
		user, err := gothic.CompleteUserAuth(res, req)
		if err != nil {
			fmt.Fprintf(res, "CompleteUserAuth failed: %v\n", err)
			return
		}

		// Save user to our session
		err = SaveUserToSession(res, req, user)
		if err != nil {
			fmt.Fprintf(res, "Failed to save user to session: %v\n", err)
			return
		}

		fmt.Println("User authenticated:", user.Email)

		// Redirect to dashboard or home page
		http.Redirect(res, req, "/dashboard", http.StatusSeeOther)
	})

	// Logout handler - wrap with GetProvider middleware
	r.With(GetProvider).Get("/logout/{provider}", func(res http.ResponseWriter, req *http.Request) {
		// Clear our user session
		err := ClearUserSession(res, req)
		if err != nil {
			fmt.Fprintf(res, "Failed to clear session: %v\n", err)
		}

		// Clear Gothic session
		err = gothic.Logout(res, req)
		if err != nil {
			fmt.Fprintf(res, "Logout failed: %v\n", err)
			return
		}

		http.Redirect(res, req, "/", http.StatusTemporaryRedirect)
	})

	// Public route - login page
	r.Get("/login", func(res http.ResponseWriter, req *http.Request) {
		t, _ := template.New("foo").Parse(indexTemplate)
		t.Execute(res, providerIndex)
	})

	// Protected route example - dashboard
	r.With(RequireAuth).Get("/dashboard", func(res http.ResponseWriter, req *http.Request) {
		// Get user from context (set by RequireAuth middleware)
		user, err := GetUserFromContext(req)
		if err != nil {
			http.Error(res, "Failed to get user", http.StatusInternalServerError)
			return
		}

		res.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(res, `
            <h1>Welcome to your Dashboard, %s!</h1>
            <p>Email: %s</p>
            <p>Provider: %s</p>
            <img src="%s" alt="Avatar" style="width:100px;border-radius:50%%">
            <p><a href="/profile">View Profile</a></p>
            <p><a href="/logout/%s">Logout</a></p>
        `, user.Name, user.Email, user.Provider, user.AvatarURL, user.Provider)
	})

	// Another protected route - profile
	r.With(RequireAuth).Get("/profile", func(res http.ResponseWriter, req *http.Request) {
		// Get user from context
		user, err := GetUserFromContext(req)
		if err != nil {
			http.Error(res, "Failed to get user", http.StatusInternalServerError)
			return
		}

		res.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(res, `
            <h1>Profile</h1>
            <p>ID: %s</p>
            <p>Name: %s %s</p>
            <p>Email: %s</p>
            <p>Provider: %s</p>
            <img src="%s" alt="Avatar" style="width:150px;border-radius:50%%">
            <p><a href="/dashboard">Back to Dashboard</a></p>
            <p><a href="/logout/%s">Logout</a></p>
        `, user.ID, user.FirstName, user.LastName, user.Email, user.Provider, user.AvatarURL, user.Provider)
	})

	// API endpoint example - returns JSON
	r.With(RequireAuth).Get("/api/me", func(res http.ResponseWriter, req *http.Request) {
		user, err := GetUserFromContext(req)
		if err != nil {
			http.Error(res, "Failed to get user", http.StatusInternalServerError)
			return
		}

		res.Header().Set("Content-Type", "application/json")
		json.NewEncoder(res).Encode(user)
	})
}
