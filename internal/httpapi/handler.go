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
//	@Summary		Cria o perfil da conta autenticada
//	@Description	A conta vem da claim "sub" do token, nunca do corpo — assim ninguém cria perfil no lugar de outra pessoa.
//	@Tags			perfil
//	@Accept			json
//	@Produce		json
//	@Param			perfil	body		domain.DadosPerfil	true	"Dados do perfil"
//	@Success		201		{object}	domain.Client
//	@Failure		400		{object}	ErroResposta	"Campos inválidos"
//	@Failure		401		{object}	ErroResposta	"Token ausente ou inválido"
//	@Failure		409		{object}	ErroResposta	"A conta já possui perfil"
//	@Security		BearerAuth
//	@Router			/clients [post]
func (h *Handler) criarPerfil(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.contaAutenticada(w, r)
	if !ok {
		return
	}

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

// meuPerfil devolve o perfil da conta autenticada.
//
//	@Summary		Perfil da conta autenticada
//	@Tags			perfil
//	@Produce		json
//	@Success		200	{object}	domain.Client
//	@Failure		401	{object}	ErroResposta	"Token ausente ou inválido"
//	@Failure		404	{object}	ErroResposta	"A conta ainda não tem perfil"
//	@Security		BearerAuth
//	@Router			/clients/me [get]
func (h *Handler) meuPerfil(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.contaAutenticada(w, r)
	if !ok {
		return
	}

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

// atualizarMeuPerfil substitui os dados do perfil da conta autenticada.
//
//	@Summary		Atualiza o próprio perfil
//	@Tags			perfil
//	@Accept			json
//	@Produce		json
//	@Param			perfil	body		domain.DadosPerfil	true	"Dados do perfil"
//	@Success		200		{object}	domain.Client
//	@Failure		400		{object}	ErroResposta	"Campos inválidos"
//	@Failure		401		{object}	ErroResposta	"Token ausente ou inválido"
//	@Failure		404		{object}	ErroResposta	"A conta ainda não tem perfil"
//	@Security		BearerAuth
//	@Router			/clients/me [put]
func (h *Handler) atualizarMeuPerfil(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.contaAutenticada(w, r)
	if !ok {
		return
	}

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

// removerMeuPerfil apaga o perfil da conta autenticada.
//
//	@Summary		Remove o próprio perfil
//	@Description	A conta em si continua existindo no Authorization Service.
//	@Tags			perfil
//	@Success		204	"Removido (ou já não existia)"
//	@Failure		401	{object}	ErroResposta	"Token ausente ou inválido"
//	@Security		BearerAuth
//	@Router			/clients/me [delete]
func (h *Handler) removerMeuPerfil(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.contaAutenticada(w, r)
	if !ok {
		return
	}

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
//
//	@Summary		Lista todos os perfis
//	@Description	Restrito a contas com papel ADMIN.
//	@Tags			admin
//	@Produce		json
//	@Success		200	{array}		domain.Client
//	@Failure		401	{object}	ErroResposta	"Token ausente ou inválido"
//	@Failure		403	{object}	ErroResposta	"A conta não é ADMIN"
//	@Security		BearerAuth
//	@Router			/clients [get]
func (h *Handler) listarPerfis(w http.ResponseWriter, r *http.Request) {
	clientes, err := h.repo.Listar(r.Context())
	if err != nil {
		h.erroInterno(w, "listando perfis", err)
		return
	}
	EscreveJSON(w, http.StatusOK, clientes)
}

// buscarPerfil é restrito a ADMIN (ver rotas).
//
//	@Summary		Busca um perfil por id
//	@Description	Restrito a contas com papel ADMIN.
//	@Tags			admin
//	@Produce		json
//	@Param			id	path		string	true	"Id do perfil (UUID)"
//	@Success		200	{object}	domain.Client
//	@Failure		400	{object}	ErroResposta	"Id não é um UUID válido"
//	@Failure		401	{object}	ErroResposta	"Token ausente ou inválido"
//	@Failure		403	{object}	ErroResposta	"A conta não é ADMIN"
//	@Failure		404	{object}	ErroResposta	"Perfil nao encontrado"
//	@Security		BearerAuth
//	@Router			/clients/{id} [get]
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

// saude responde ao healthcheck.
//
//	@Summary	Health check
//	@Tags		infra
//	@Produce	json
//	@Success	200	{object}	map[string]string
//	@Router		/health [get]
func (h *Handler) saude(w http.ResponseWriter, _ *http.Request) {
	EscreveJSON(w, http.StatusOK, map[string]string{"status": "UP"})
}

// contaAutenticada recupera as claims postas pelo middleware.
//
// Só devolve ok=false se a rota tiver sido registrada sem o middleware de
// autenticação — erro de programação, não de quem chamou. Sem esta checagem
// seria um acesso a ponteiro nulo.
func (h *Handler) contaAutenticada(w http.ResponseWriter, r *http.Request) (*auth.Claims, bool) {
	claims, ok := auth.ClaimsDo(r.Context())
	if !ok {
		h.log.Error("rota protegida registrada sem o middleware de autenticação",
			"rota", r.URL.Path)
		EscreveErro(w, http.StatusUnauthorized, "Autenticação necessária.")
		return nil, false
	}
	return claims, true
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
