package httpapi_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/google/uuid"

	"github.com/Back-End-Cloud-Computing/client/internal/domain"
	"github.com/Back-End-Cloud-Computing/client/internal/httpapi"
)

// repoQuePanica simula uma falha inesperada na camada de dados.
type repoQuePanica struct{ repoFalso }

func (r *repoQuePanica) BuscarPorConta(context.Context, uuid.UUID) (*domain.Client, error) {
	panic("falha inesperada no repositório")
}

// Um panic precisa virar 500 no formato de erro da API. Sem o middleware de
// recuperação, o net/http derruba a conexão e quem chamou vê erro de rede em
// vez de erro do serviço.
func TestPanicViraErro500(t *testing.T) {
	amb := montarCom(t, &repoQuePanica{*novoRepoFalso()})

	resposta := amb.requisicao(t, http.MethodGet, "/clients/me",
		amb.token(t, uuid.New(), "CLIENTE"), "")

	if resposta.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status = %d, esperado 500", resposta.StatusCode)
	}

	corpo, err := io.ReadAll(resposta.Body)
	if err != nil {
		t.Fatalf("lendo corpo: %v", err)
	}

	var erro httpapi.ErroResposta
	if err := json.Unmarshal(corpo, &erro); err != nil {
		t.Fatalf("a resposta não é o JSON de erro da API: %v (corpo: %q)", err, corpo)
	}
	if erro.Status != http.StatusInternalServerError {
		t.Errorf("status no corpo = %d", erro.Status)
	}
}

// O Vary: Origin precisa sair mesmo quando a origem não é permitida — senão um
// cache compartilhado poderia servir a uma origem a resposta guardada de outra.
func TestVaryOriginSemprePresente(t *testing.T) {
	amb := montar(t)

	req, err := http.NewRequest(http.MethodGet, amb.servidor.URL+"/health", nil)
	if err != nil {
		t.Fatalf("montando requisição: %v", err)
	}
	req.Header.Set("Origin", "http://site-nao-autorizado.com")

	resposta, err := amb.servidor.Client().Do(req)
	if err != nil {
		t.Fatalf("requisição: %v", err)
	}
	defer resposta.Body.Close()

	if vary := resposta.Header.Get("Vary"); vary != "Origin" {
		t.Errorf("Vary = %q, esperado \"Origin\"", vary)
	}
	if origem := resposta.Header.Get("Access-Control-Allow-Origin"); origem != "" {
		t.Errorf("origem não permitida recebeu Allow-Origin = %q", origem)
	}
}
