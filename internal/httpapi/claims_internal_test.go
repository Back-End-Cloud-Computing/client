package httpapi

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Teste interno: alcança o handler sem passar pelo middleware, o que um teste
// externo não conseguiria fazer.
//
// Uma rota protegida registrada por engano sem o middleware de autenticação
// não pode derrubar o serviço — antes, o handler acessava as claims sem
// conferir se existiam.
func TestHandlerSemMiddlewareRespondeSemPanicar(t *testing.T) {
	h := NewHandler(nil, slog.New(slog.NewTextHandler(io.Discard, nil)))

	casos := map[string]http.HandlerFunc{
		"criarPerfil":        h.criarPerfil,
		"meuPerfil":          h.meuPerfil,
		"atualizarMeuPerfil": h.atualizarMeuPerfil,
		"removerMeuPerfil":   h.removerMeuPerfil,
	}

	for nome, handler := range casos {
		t.Run(nome, func(t *testing.T) {
			defer func() {
				if motivo := recover(); motivo != nil {
					t.Fatalf("o handler entrou em panic sem as claims: %v", motivo)
				}
			}()

			gravador := httptest.NewRecorder()
			handler(gravador, httptest.NewRequest(http.MethodGet, "/qualquer", nil))

			if gravador.Code != http.StatusUnauthorized {
				t.Errorf("status = %d, esperado 401", gravador.Code)
			}
		})
	}
}
