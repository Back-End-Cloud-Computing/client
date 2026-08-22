package auth

import (
	"context"
	"net/http"
	"strings"
)

type contextKey string

const claimsKey contextKey = "ganjj.claims"

// ErrorWriter escreve a resposta de erro no formato da API.
// Injetado para o pacote auth não depender do pacote de HTTP.
type ErrorWriter func(w http.ResponseWriter, status int, mensagem string)

// Requer exige um token de acesso válido.
func (v *Verifier) Requer(escreveErro ErrorWriter) func(http.Handler) http.Handler {
	return func(proximo http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := tokenDoHeader(r)
			if token == "" {
				escreveErro(w, http.StatusUnauthorized,
					"Autenticação necessária. Envie um token em Authorization: Bearer.")
				return
			}

			claims, err := v.Verify(token)
			if err != nil {
				mensagem := "Token inválido."
				if err == ErrTokenExpirado {
					mensagem = "Token expirado. Renove em POST /auth/refresh no Authorization Service."
				}
				escreveErro(w, http.StatusUnauthorized, mensagem)
				return
			}

			proximo.ServeHTTP(w, r.WithContext(ComClaims(r.Context(), claims)))
		})
	}
}

// RequerAdmin exige, além do token válido, o papel ADMIN.
func RequerAdmin(escreveErro ErrorWriter) func(http.Handler) http.Handler {
	return func(proximo http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := ClaimsDo(r.Context())
			if !ok || !claims.IsAdmin() {
				escreveErro(w, http.StatusForbidden,
					"Esta conta não tem permissão para acessar este recurso.")
				return
			}
			proximo.ServeHTTP(w, r)
		})
	}
}

// ComClaims guarda as claims no contexto da requisição.
func ComClaims(ctx context.Context, claims *Claims) context.Context {
	return context.WithValue(ctx, claimsKey, claims)
}

// ClaimsDo recupera as claims colocadas pelo middleware.
func ClaimsDo(ctx context.Context) (*Claims, bool) {
	claims, ok := ctx.Value(claimsKey).(*Claims)
	return claims, ok
}

func tokenDoHeader(r *http.Request) string {
	header := r.Header.Get("Authorization")
	const prefixo = "Bearer "
	if !strings.HasPrefix(header, prefixo) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(header, prefixo))
}
