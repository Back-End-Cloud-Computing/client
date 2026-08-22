package httpapi_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/Back-End-Cloud-Computing/client/internal/auth"
	"github.com/Back-End-Cloud-Computing/client/internal/domain"
	"github.com/Back-End-Cloud-Computing/client/internal/httpapi"
)

const emissor = "ganjj-authorization"

// --- repositório falso -------------------------------------------------

type repoFalso struct {
	porConta map[uuid.UUID]*domain.Client
}

func novoRepoFalso() *repoFalso {
	return &repoFalso{porConta: map[uuid.UUID]*domain.Client{}}
}

func (r *repoFalso) Criar(_ context.Context, conta uuid.UUID, dados domain.DadosPerfil) (*domain.Client, error) {
	if _, existe := r.porConta[conta]; existe {
		return nil, domain.ErrPerfilJaExiste
	}
	cliente := &domain.Client{
		ID: uuid.New(), AccountID: conta, Nome: dados.Nome, Telefone: dados.Telefone,
		Endereco: dados.Endereco, CriadoEm: time.Now(), AtualizadoEm: time.Now(),
	}
	r.porConta[conta] = cliente
	return cliente, nil
}

func (r *repoFalso) BuscarPorConta(_ context.Context, conta uuid.UUID) (*domain.Client, error) {
	if cliente, ok := r.porConta[conta]; ok {
		return cliente, nil
	}
	return nil, domain.ErrPerfilNaoEncontrado
}

func (r *repoFalso) BuscarPorID(_ context.Context, id uuid.UUID) (*domain.Client, error) {
	for _, cliente := range r.porConta {
		if cliente.ID == id {
			return cliente, nil
		}
	}
	return nil, domain.ErrPerfilNaoEncontrado
}

func (r *repoFalso) Atualizar(_ context.Context, conta uuid.UUID, dados domain.DadosPerfil) (*domain.Client, error) {
	cliente, ok := r.porConta[conta]
	if !ok {
		return nil, domain.ErrPerfilNaoEncontrado
	}
	cliente.Nome = dados.Nome
	cliente.Telefone = dados.Telefone
	cliente.Endereco = dados.Endereco
	cliente.AtualizadoEm = time.Now()
	return cliente, nil
}

func (r *repoFalso) Listar(_ context.Context) ([]domain.Client, error) {
	clientes := []domain.Client{}
	for _, cliente := range r.porConta {
		clientes = append(clientes, *cliente)
	}
	return clientes, nil
}

func (r *repoFalso) Remover(_ context.Context, conta uuid.UUID) error {
	if _, ok := r.porConta[conta]; !ok {
		return domain.ErrPerfilNaoEncontrado
	}
	delete(r.porConta, conta)
	return nil
}

// --- apoio -------------------------------------------------------------

type ambiente struct {
	servidor *httptest.Server
	chave    *rsa.PrivateKey
}

func montar(t *testing.T) *ambiente {
	t.Helper()
	return montarCom(t, novoRepoFalso())
}

// montarCom permite trocar o repositório — usado para simular falhas.
func montarCom(t *testing.T, repo httpapi.Repositorio) *ambiente {
	t.Helper()

	chave, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("gerando chave: %v", err)
	}

	der, _ := x509.MarshalPKIXPublicKey(&chave.PublicKey)
	caminho := filepath.Join(t.TempDir(), "public.pem")
	if err := os.WriteFile(caminho,
		pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}), 0o600); err != nil {
		t.Fatalf("gravando PEM: %v", err)
	}

	verificador, err := auth.NewVerifier(caminho, emissor)
	if err != nil {
		t.Fatalf("criando verificador: %v", err)
	}

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := httpapi.Router(httpapi.NewHandler(repo, log), verificador,
		[]string{"http://localhost:4200"})

	servidor := httptest.NewServer(router)
	t.Cleanup(servidor.Close)

	return &ambiente{servidor: servidor, chave: chave}
}

func (a *ambiente) token(t *testing.T, conta uuid.UUID, papel string) string {
	t.Helper()
	agora := time.Now()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"iss": emissor, "sub": conta.String(), "email": "cliente@ganjj.com",
		"role": papel, "typ": "access",
		"iat": agora.Unix(), "exp": agora.Add(15 * time.Minute).Unix(),
	})
	assinado, err := token.SignedString(a.chave)
	if err != nil {
		t.Fatalf("assinando token: %v", err)
	}
	return assinado
}

func (a *ambiente) requisicao(t *testing.T, metodo, caminho, token, corpo string) *http.Response {
	t.Helper()

	var leitor io.Reader
	if corpo != "" {
		leitor = strings.NewReader(corpo)
	}

	req, err := http.NewRequest(metodo, a.servidor.URL+caminho, leitor)
	if err != nil {
		t.Fatalf("montando requisição: %v", err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if corpo != "" {
		req.Header.Set("Content-Type", "application/json")
	}

	resposta, err := a.servidor.Client().Do(req)
	if err != nil {
		t.Fatalf("executando requisição: %v", err)
	}
	t.Cleanup(func() { resposta.Body.Close() })
	return resposta
}

const perfilValido = `{
	"nome": "Eduardo Fabri",
	"telefone": "(11) 98765-4321",
	"endereco": {
		"logradouro": "Rua das Flores", "numero": "123", "bairro": "Centro",
		"cidade": "São Paulo", "estado": "sp", "cep": "01310-100"
	}
}`

// --- testes ------------------------------------------------------------

func TestCriaEBuscaPerfil(t *testing.T) {
	amb := montar(t)
	conta := uuid.New()
	token := amb.token(t, conta, "CLIENTE")

	resposta := amb.requisicao(t, http.MethodPost, "/clients", token, perfilValido)
	if resposta.StatusCode != http.StatusCreated {
		t.Fatalf("POST /clients = %d, esperado 201", resposta.StatusCode)
	}

	var criado domain.Client
	if err := json.NewDecoder(resposta.Body).Decode(&criado); err != nil {
		t.Fatalf("decodificando resposta: %v", err)
	}

	// A conta vem do token, nunca do corpo.
	if criado.AccountID != conta {
		t.Errorf("accountId = %v, esperado %v", criado.AccountID, conta)
	}
	// Normalização: a sigla do estado veio "sp" e deve ser gravada "SP".
	if criado.Endereco.Estado != "SP" {
		t.Errorf("estado = %q, esperado \"SP\"", criado.Endereco.Estado)
	}

	resposta = amb.requisicao(t, http.MethodGet, "/clients/me", token, "")
	if resposta.StatusCode != http.StatusOK {
		t.Fatalf("GET /clients/me = %d, esperado 200", resposta.StatusCode)
	}
}

func TestCadaContaEnxergaSomenteOProprioPerfil(t *testing.T) {
	amb := montar(t)

	tokenA := amb.token(t, uuid.New(), "CLIENTE")
	amb.requisicao(t, http.MethodPost, "/clients", tokenA, perfilValido)

	// Outra conta, sem perfil criado.
	tokenB := amb.token(t, uuid.New(), "CLIENTE")
	resposta := amb.requisicao(t, http.MethodGet, "/clients/me", tokenB, "")
	if resposta.StatusCode != http.StatusNotFound {
		t.Fatalf("GET /clients/me de outra conta = %d, esperado 404", resposta.StatusCode)
	}
}

func TestNaoPermiteDoisPerfisParaAMesmaConta(t *testing.T) {
	amb := montar(t)
	token := amb.token(t, uuid.New(), "CLIENTE")

	amb.requisicao(t, http.MethodPost, "/clients", token, perfilValido)
	resposta := amb.requisicao(t, http.MethodPost, "/clients", token, perfilValido)

	if resposta.StatusCode != http.StatusConflict {
		t.Fatalf("segundo POST = %d, esperado 409", resposta.StatusCode)
	}
}

func TestAtualizaPerfil(t *testing.T) {
	amb := montar(t)
	token := amb.token(t, uuid.New(), "CLIENTE")
	amb.requisicao(t, http.MethodPost, "/clients", token, perfilValido)

	atualizado := strings.Replace(perfilValido, "Eduardo Fabri", "Eduardo H. Fabri", 1)
	resposta := amb.requisicao(t, http.MethodPut, "/clients/me", token, atualizado)
	if resposta.StatusCode != http.StatusOK {
		t.Fatalf("PUT /clients/me = %d, esperado 200", resposta.StatusCode)
	}

	var cliente domain.Client
	json.NewDecoder(resposta.Body).Decode(&cliente)
	if cliente.Nome != "Eduardo H. Fabri" {
		t.Errorf("nome = %q", cliente.Nome)
	}
}

func TestRemovePerfil(t *testing.T) {
	amb := montar(t)
	token := amb.token(t, uuid.New(), "CLIENTE")
	amb.requisicao(t, http.MethodPost, "/clients", token, perfilValido)

	resposta := amb.requisicao(t, http.MethodDelete, "/clients/me", token, "")
	if resposta.StatusCode != http.StatusNoContent {
		t.Fatalf("DELETE = %d, esperado 204", resposta.StatusCode)
	}

	resposta = amb.requisicao(t, http.MethodGet, "/clients/me", token, "")
	if resposta.StatusCode != http.StatusNotFound {
		t.Fatalf("GET após remover = %d, esperado 404", resposta.StatusCode)
	}
}

func TestValidacaoDeCampos(t *testing.T) {
	amb := montar(t)
	token := amb.token(t, uuid.New(), "CLIENTE")

	invalido := `{
		"nome": "Ed",
		"telefone": "abc",
		"endereco": {"logradouro": "", "numero": "", "cidade": "", "estado": "paulista", "cep": "123"}
	}`

	resposta := amb.requisicao(t, http.MethodPost, "/clients", token, invalido)
	if resposta.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, esperado 400", resposta.StatusCode)
	}

	var erro httpapi.ErroResposta
	json.NewDecoder(resposta.Body).Decode(&erro)

	for _, campo := range []string{"nome", "telefone", "endereco.cep", "endereco.estado"} {
		if _, ok := erro.Campos[campo]; !ok {
			t.Errorf("faltou o erro do campo %q", campo)
		}
	}
}

func TestExigeAutenticacao(t *testing.T) {
	amb := montar(t)

	casos := []struct{ metodo, caminho string }{
		{http.MethodPost, "/clients"},
		{http.MethodGet, "/clients/me"},
		{http.MethodPut, "/clients/me"},
		{http.MethodDelete, "/clients/me"},
		{http.MethodGet, "/clients"},
	}

	for _, caso := range casos {
		resposta := amb.requisicao(t, caso.metodo, caso.caminho, "", "")
		if resposta.StatusCode != http.StatusUnauthorized {
			t.Errorf("%s %s = %d, esperado 401", caso.metodo, caso.caminho, resposta.StatusCode)
		}
	}
}

func TestRotasAdministrativas(t *testing.T) {
	amb := montar(t)

	tokenCliente := amb.token(t, uuid.New(), "CLIENTE")
	amb.requisicao(t, http.MethodPost, "/clients", tokenCliente, perfilValido)

	resposta := amb.requisicao(t, http.MethodGet, "/clients", tokenCliente, "")
	if resposta.StatusCode != http.StatusForbidden {
		t.Fatalf("CLIENTE em GET /clients = %d, esperado 403", resposta.StatusCode)
	}

	tokenAdmin := amb.token(t, uuid.New(), "ADMIN")
	resposta = amb.requisicao(t, http.MethodGet, "/clients", tokenAdmin, "")
	if resposta.StatusCode != http.StatusOK {
		t.Fatalf("ADMIN em GET /clients = %d, esperado 200", resposta.StatusCode)
	}

	var clientes []domain.Client
	json.NewDecoder(resposta.Body).Decode(&clientes)
	if len(clientes) != 1 {
		t.Fatalf("listou %d perfis, esperado 1", len(clientes))
	}
}

func TestBuscaPorIDSoParaAdmin(t *testing.T) {
	amb := montar(t)

	tokenCliente := amb.token(t, uuid.New(), "CLIENTE")
	resposta := amb.requisicao(t, http.MethodPost, "/clients", tokenCliente, perfilValido)

	var criado domain.Client
	json.NewDecoder(resposta.Body).Decode(&criado)

	resposta = amb.requisicao(t, http.MethodGet, "/clients/"+criado.ID.String(), tokenCliente, "")
	if resposta.StatusCode != http.StatusForbidden {
		t.Fatalf("CLIENTE = %d, esperado 403", resposta.StatusCode)
	}

	tokenAdmin := amb.token(t, uuid.New(), "ADMIN")
	resposta = amb.requisicao(t, http.MethodGet, "/clients/"+criado.ID.String(), tokenAdmin, "")
	if resposta.StatusCode != http.StatusOK {
		t.Fatalf("ADMIN = %d, esperado 200", resposta.StatusCode)
	}

	resposta = amb.requisicao(t, http.MethodGet, "/clients/nao-e-uuid", tokenAdmin, "")
	if resposta.StatusCode != http.StatusBadRequest {
		t.Fatalf("id inválido = %d, esperado 400", resposta.StatusCode)
	}
}

func TestHealthNaoExigeToken(t *testing.T) {
	amb := montar(t)
	resposta := amb.requisicao(t, http.MethodGet, "/health", "", "")
	if resposta.StatusCode != http.StatusOK {
		t.Fatalf("GET /health = %d, esperado 200", resposta.StatusCode)
	}
}

func TestCORS(t *testing.T) {
	amb := montar(t)

	req, _ := http.NewRequest(http.MethodOptions, amb.servidor.URL+"/clients/me", nil)
	req.Header.Set("Origin", "http://localhost:4200")
	req.Header.Set("Access-Control-Request-Method", "GET")

	resposta, err := amb.servidor.Client().Do(req)
	if err != nil {
		t.Fatalf("preflight: %v", err)
	}
	defer resposta.Body.Close()

	if resposta.StatusCode != http.StatusOK {
		t.Fatalf("preflight = %d, esperado 200", resposta.StatusCode)
	}
	if origem := resposta.Header.Get("Access-Control-Allow-Origin"); origem != "http://localhost:4200" {
		t.Errorf("Allow-Origin = %q", origem)
	}

	req, _ = http.NewRequest(http.MethodOptions, amb.servidor.URL+"/clients/me", nil)
	req.Header.Set("Origin", "http://site-nao-autorizado.com")
	req.Header.Set("Access-Control-Request-Method", "GET")

	resposta, err = amb.servidor.Client().Do(req)
	if err != nil {
		t.Fatalf("preflight de origem desconhecida: %v", err)
	}
	defer resposta.Body.Close()

	if resposta.StatusCode != http.StatusForbidden {
		t.Errorf("origem desconhecida = %d, esperado 403", resposta.StatusCode)
	}
}
