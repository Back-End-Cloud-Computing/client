package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	"github.com/Back-End-Cloud-Computing/client/internal/auth"
	"github.com/Back-End-Cloud-Computing/client/internal/domain"
)

// Repositorio é o que o handler precisa do armazenamento.
type Repositorio interface {
	Criar(ctx context.Context, accountID uuid.UUID, dados domain.DadosPerfil) (*domain.Client, error)
	BuscarPorConta(ctx context.Context, accountID uuid.UUID) (*domain.Client, error)
	BuscarPorID(ctx context.Context, id uuid.UUID) (*domain.Client, error)
	Atualizar(ctx context.Context, accountID uuid.UUID, dados domain.DadosPerfil) (*domain.Client, error)
	Listar(ctx context.Context) ([]domain.Client, error)
	Remover(ctx context.Context, accountID uuid.UUID) error
}

type Handler struct {
	repo Repositorio
	log  *slog.Logger
}

func NewHandler(repo Repositorio, log *slog.Logger) *Handler {
	return &Handler{repo: repo, log: log}
}

// criarPerfil cria o perfil da conta autenticada.
//
// A conta vem do token, nunca do corpo da requisição: assim ninguém consegue
// criar um perfil no lugar de outra pessoa.
func (h *Handler) criarPerfil(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.ClaimsDo(r.Context())

	dados, ok := h.lerDados(w, r)
	if !ok {
		return
	}

	cliente, err := h.repo.Criar(r.Context(), claims.AccountID, dados)
	if errors.Is(err, domain.ErrPerfilJaExiste) {
		EscreveErro(w, http.StatusConflict, "Esta conta já possui um perfil.")
		return
	}
	if err != nil {
		h.erroInterno(w, "criando perfil", err)
		return
	}

	EscreveJSON(w, http.StatusCreated, cliente)
}

func (h *Handler) meuPerfil(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.ClaimsDo(r.Context())

	cliente, err := h.repo.BuscarPorConta(r.Context(), claims.AccountID)
	if errors.Is(err, domain.ErrPerfilNaoEncontrado) {
		EscreveErro(w, http.StatusNotFound,
			"Esta conta ainda não tem perfil. Crie um em POST /clients.")
		return
	}
	if err != nil {
		h.erroInterno(w, "buscando perfil", err)
		return
	}

	EscreveJSON(w, http.StatusOK, cliente)
}

func (h *Handler) atualizarMeuPerfil(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.ClaimsDo(r.Context())

	dados, ok := h.lerDados(w, r)
	if !ok {
		return
	}

	cliente, err := h.repo.Atualizar(r.Context(), claims.AccountID, dados)
	if errors.Is(err, domain.ErrPerfilNaoEncontrado) {
		EscreveErro(w, http.StatusNotFound,
			"Esta conta ainda não tem perfil. Crie um em POST /clients.")
		return
	}
	if err != nil {
		h.erroInterno(w, "atualizando perfil", err)
		return
	}

	EscreveJSON(w, http.StatusOK, cliente)
}

func (h *Handler) removerMeuPerfil(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.ClaimsDo(r.Context())

	err := h.repo.Remover(r.Context(), claims.AccountID)
	if errors.Is(err, domain.ErrPerfilNaoEncontrado) {
		// Já não existe: o resultado desejado já vale.
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if err != nil {
		h.erroInterno(w, "removendo perfil", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// listarPerfis é restrito a ADMIN (ver rotas).
func (h *Handler) listarPerfis(w http.ResponseWriter, r *http.Request) {
	clientes, err := h.repo.Listar(r.Context())
	if err != nil {
		h.erroInterno(w, "listando perfis", err)
		return
	}
	EscreveJSON(w, http.StatusOK, clientes)
}

// buscarPerfil é restrito a ADMIN (ver rotas).
func (h *Handler) buscarPerfil(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		EscreveErro(w, http.StatusBadRequest, "O id informado não é um UUID válido.")
		return
	}

	cliente, err := h.repo.BuscarPorID(r.Context(), id)
	if errors.Is(err, domain.ErrPerfilNaoEncontrado) {
		EscreveErro(w, http.StatusNotFound, "Perfil não encontrado.")
		return
	}
	if err != nil {
		h.erroInterno(w, "buscando perfil por id", err)
		return
	}

	EscreveJSON(w, http.StatusOK, cliente)
}

func (h *Handler) saude(w http.ResponseWriter, _ *http.Request) {
	EscreveJSON(w, http.StatusOK, map[string]string{"status": "UP"})
}

// lerDados decodifica e valida o corpo. Devolve ok=false se já respondeu erro.
func (h *Handler) lerDados(w http.ResponseWriter, r *http.Request) (domain.DadosPerfil, bool) {
	var dados domain.DadosPerfil

	decodificador := json.NewDecoder(r.Body)
	decodificador.DisallowUnknownFields()

	if err := decodificador.Decode(&dados); err != nil {
		EscreveErro(w, http.StatusBadRequest, "Corpo da requisição inválido.")
		return dados, false
	}

	dados.Normalizar()

	if erros := dados.Validar(); len(erros) > 0 {
		escreveErroValidacao(w, erros)
		return dados, false
	}

	return dados, true
}

func (h *Handler) erroInterno(w http.ResponseWriter, acao string, err error) {
	h.log.Error("falha inesperada", "acao", acao, "erro", err)
	EscreveErro(w, http.StatusInternalServerError,
		"Não foi possível concluir a operação. Tente novamente.")
}
