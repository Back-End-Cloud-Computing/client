// Package httpapi expõe o serviço por HTTP.
package httpapi

import (
	"encoding/json"
	"net/http"
	"time"
)

// ErroResposta segue o mesmo formato do Authorization Service, para o frontend
// tratar erro de um jeito só.
type ErroResposta struct {
	Status    int               `json:"status"`
	Erro      string            `json:"error"`
	Mensagem  string            `json:"message"`
	Campos    map[string]string `json:"fields,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
}

func EscreveJSON(w http.ResponseWriter, status int, corpo any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if corpo != nil {
		_ = json.NewEncoder(w).Encode(corpo)
	}
}

func EscreveErro(w http.ResponseWriter, status int, mensagem string) {
	EscreveJSON(w, status, ErroResposta{
		Status:    status,
		Erro:      http.StatusText(status),
		Mensagem:  mensagem,
		Timestamp: time.Now(),
	})
}

func escreveErroValidacao(w http.ResponseWriter, campos map[string]string) {
	EscreveJSON(w, http.StatusBadRequest, ErroResposta{
		Status:    http.StatusBadRequest,
		Erro:      "Validation Failed",
		Mensagem:  "Há campos inválidos na requisição.",
		Campos:    campos,
		Timestamp: time.Now(),
	})
}
