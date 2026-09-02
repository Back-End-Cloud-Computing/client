package httpapi

import (
	"log/slog"
	"net/http"
	"runtime/debug"
	"slices"
	"strings"

	httpSwagger "github.com/swaggo/http-swagger/v2"

	_ "github.com/Back-End-Cloud-Computing/client/docs" // registra a spec gerada pelo swag
	"github.com/Back-End-Cloud-Computing/client/internal/auth"
)

// Router monta as rotas do serviço.
//
// Usa o roteamento nativo do net/http (Go 1.22+), com padrões do tipo
// "GET /clients/{id}" — sem dependência de framework.
func Router(h *Handler, verificador *auth.Verifier, origensPermitidas []string) http.Handler {
	mux := http.NewServeMux()

	protegido := verificador.Requer(EscreveErro)
	somenteAdmin := auth.RequerAdmin(EscreveErro)

	// Sem autenticação: usado pelo healthcheck do Docker.
	mux.HandleFunc("GET /health", h.saude)

	// Sem autenticação: mostra qual réplica respondeu, para observar a
	// distribuição entre Pods no Kubernetes.
	mux.HandleFunc("GET /instance", h.instancia)

	// Documentação interativa. A UI é embutida no binário, então funciona sem
	// internet. Em /swagger/index.html.
	mux.Handle("GET /swagger/", httpSwagger.WrapHandler)

	// Perfil da própria conta.
	mux.Handle("POST /clients", protegido(http.HandlerFunc(h.criarPerfil)))
	mux.Handle("GET /clients/me", protegido(http.HandlerFunc(h.meuPerfil)))
	mux.Handle("PUT /clients/me", protegido(http.HandlerFunc(h.atualizarMeuPerfil)))
	mux.Handle("DELETE /clients/me", protegido(http.HandlerFunc(h.removerMeuPerfil)))

	// Rotas administrativas: exigem token válido e papel ADMIN.
	mux.Handle("GET /clients", protegido(somenteAdmin(http.HandlerFunc(h.listarPerfis))))
	mux.Handle("GET /clients/{id}", protegido(somenteAdmin(http.HandlerFunc(h.buscarPerfil))))

	return recuperaPanico(h.log)(cors(origensPermitidas)(mux))
}

// recuperaPanico transforma um panic em 500 no formato de erro da API.
//
// Sem isso o net/http derruba a conexão sem resposta nenhuma, e quem chamou vê
// um erro de rede em vez de um erro do serviço.
func recuperaPanico(log *slog.Logger) func(http.Handler) http.Handler {
	return func(proximo http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if motivo := recover(); motivo != nil {
					log.Error("panic ao tratar requisição",
						"rota", r.URL.Path, "motivo", motivo,
						"stack", string(debug.Stack()))
					EscreveErro(w, http.StatusInternalServerError,
						"Não foi possível concluir a operação. Tente novamente.")
				}
			}()
			proximo.ServeHTTP(w, r)
		})
	}
}

// cors responde ao preflight e libera apenas as origens configuradas.
func cors(origensPermitidas []string) func(http.Handler) http.Handler {
	return func(proximo http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origem := r.Header.Get("Origin")
			permitida := origem != "" && slices.Contains(origensPermitidas, origem)

			// Sempre: a resposta depende do Origin mesmo quando ele não é
			// permitido. Anunciar isso só nos casos liberados deixaria um cache
			// servir a uma origem a resposta guardada para outra.
			w.Header().Add("Vary", "Origin")

			if permitida {
				w.Header().Set("Access-Control-Allow-Origin", origem)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
				if !permitida {
					w.WriteHeader(http.StatusForbidden)
					return
				}
				w.Header().Set("Access-Control-Allow-Methods",
					strings.Join([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}, ", "))
				w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
				w.Header().Set("Access-Control-Max-Age", "3600")
				w.WriteHeader(http.StatusOK)
				return
			}

			proximo.ServeHTTP(w, r)
		})
	}
}
