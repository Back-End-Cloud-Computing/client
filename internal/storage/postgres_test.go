package storage_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/Back-End-Cloud-Computing/client/internal/domain"
	"github.com/Back-End-Cloud-Computing/client/internal/storage"
)

// Estes testes rodam contra um PostgreSQL de verdade, num container criado na
// hora. É a única forma de exercitar o SQL: um repositório falso confirmaria a
// lógica do handler, mas não diria nada sobre as queries.
//
// Precisam de Docker. Sem Docker, use "go test -short ./..." para pulá-los.

func novoBanco(t *testing.T) *pgxpool.Pool {
	t.Helper()

	if testing.Short() {
		t.Skip("pulando teste de integração (-short)")
	}

	ctx := context.Background()

	container, err := tcpostgres.Run(ctx, "postgres:17-alpine",
		tcpostgres.WithDatabase("ganjj_client_test"),
		tcpostgres.WithUsername("teste"),
		tcpostgres.WithPassword("teste"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second)),
	)
	if err != nil {
		t.Skipf("não foi possível subir o Postgres de teste (Docker está rodando?): %v", err)
	}

	t.Cleanup(func() {
		if err := testcontainers.TerminateContainer(container); err != nil {
			t.Logf("removendo container: %v", err)
		}
	})

	url, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("montando string de conexão: %v", err)
	}

	pool, err := storage.Conectar(ctx, url)
	if err != nil {
		t.Fatalf("conectando: %v", err)
	}
	t.Cleanup(pool.Close)

	if err := storage.Migrar(ctx, pool); err != nil {
		t.Fatalf("migrando: %v", err)
	}

	return pool
}

func dadosDeTeste() domain.DadosPerfil {
	return domain.DadosPerfil{
		Nome:     "Eduardo Fabri",
		Telefone: "(11) 98765-4321",
		Endereco: domain.Endereco{
			Logradouro: "Rua das Flores", Numero: "123", Complemento: "apto 45",
			Bairro: "Centro", Cidade: "São Paulo", Estado: "SP", CEP: "01310-100",
		},
	}
}

func TestCriarEBuscar(t *testing.T) {
	repo := storage.NewClientRepository(novoBanco(t))
	ctx := context.Background()
	conta := uuid.New()

	criado, err := repo.Criar(ctx, conta, dadosDeTeste())
	if err != nil {
		t.Fatalf("criando: %v", err)
	}

	if criado.AccountID != conta {
		t.Errorf("accountId = %v, esperado %v", criado.AccountID, conta)
	}
	if criado.ID == uuid.Nil {
		t.Error("o id não foi gerado")
	}
	if criado.CriadoEm.IsZero() {
		t.Error("criadoEm não foi preenchido pelo banco")
	}

	buscado, err := repo.BuscarPorConta(ctx, conta)
	if err != nil {
		t.Fatalf("buscando por conta: %v", err)
	}
	if buscado.ID != criado.ID {
		t.Errorf("id = %v, esperado %v", buscado.ID, criado.ID)
	}
	// Campos opcionais precisam sobreviver à ida e volta.
	if buscado.Endereco.Complemento != "apto 45" {
		t.Errorf("complemento = %q", buscado.Endereco.Complemento)
	}
	if buscado.Endereco.Bairro != "Centro" {
		t.Errorf("bairro = %q", buscado.Endereco.Bairro)
	}

	porID, err := repo.BuscarPorID(ctx, criado.ID)
	if err != nil {
		t.Fatalf("buscando por id: %v", err)
	}
	if porID.AccountID != conta {
		t.Errorf("busca por id trouxe a conta errada")
	}
}

// O UNIQUE em account_id precisa virar um erro de domínio, não um erro cru do
// driver — é o que o handler usa para responder 409.
func TestContaDuplicadaViraErroDeDominio(t *testing.T) {
	repo := storage.NewClientRepository(novoBanco(t))
	ctx := context.Background()
	conta := uuid.New()

	if _, err := repo.Criar(ctx, conta, dadosDeTeste()); err != nil {
		t.Fatalf("primeira criação: %v", err)
	}

	_, err := repo.Criar(ctx, conta, dadosDeTeste())
	if !errors.Is(err, domain.ErrPerfilJaExiste) {
		t.Fatalf("erro = %v, esperado ErrPerfilJaExiste", err)
	}
}

func TestCamposOpcionaisVazios(t *testing.T) {
	repo := storage.NewClientRepository(novoBanco(t))
	ctx := context.Background()

	dados := dadosDeTeste()
	dados.Endereco.Complemento = ""
	dados.Endereco.Bairro = ""

	criado, err := repo.Criar(ctx, uuid.New(), dados)
	if err != nil {
		t.Fatalf("criando sem complemento/bairro: %v", err)
	}
	if criado.Endereco.Complemento != "" || criado.Endereco.Bairro != "" {
		t.Errorf("campos opcionais vazios não voltaram vazios: %+v", criado.Endereco)
	}
}

func TestAtualizar(t *testing.T) {
	repo := storage.NewClientRepository(novoBanco(t))
	ctx := context.Background()
	conta := uuid.New()

	criado, err := repo.Criar(ctx, conta, dadosDeTeste())
	if err != nil {
		t.Fatalf("criando: %v", err)
	}

	novos := dadosDeTeste()
	novos.Nome = "Eduardo H. Fabri"
	novos.Endereco.Cidade = "Campinas"

	atualizado, err := repo.Atualizar(ctx, conta, novos)
	if err != nil {
		t.Fatalf("atualizando: %v", err)
	}

	if atualizado.Nome != "Eduardo H. Fabri" {
		t.Errorf("nome = %q", atualizado.Nome)
	}
	if atualizado.Endereco.Cidade != "Campinas" {
		t.Errorf("cidade = %q", atualizado.Endereco.Cidade)
	}
	// Atualizar não pode trocar a identidade do registro.
	if atualizado.ID != criado.ID {
		t.Errorf("o id mudou: %v -> %v", criado.ID, atualizado.ID)
	}
	if !atualizado.AtualizadoEm.After(criado.AtualizadoEm) &&
		!atualizado.AtualizadoEm.Equal(criado.AtualizadoEm) {
		t.Error("atualizadoEm não avançou")
	}
}

func TestAtualizarPerfilInexistente(t *testing.T) {
	repo := storage.NewClientRepository(novoBanco(t))

	_, err := repo.Atualizar(context.Background(), uuid.New(), dadosDeTeste())
	if !errors.Is(err, domain.ErrPerfilNaoEncontrado) {
		t.Fatalf("erro = %v, esperado ErrPerfilNaoEncontrado", err)
	}
}

func TestBuscarInexistente(t *testing.T) {
	repo := storage.NewClientRepository(novoBanco(t))
	ctx := context.Background()

	if _, err := repo.BuscarPorConta(ctx, uuid.New()); !errors.Is(err, domain.ErrPerfilNaoEncontrado) {
		t.Errorf("BuscarPorConta: erro = %v, esperado ErrPerfilNaoEncontrado", err)
	}
	if _, err := repo.BuscarPorID(ctx, uuid.New()); !errors.Is(err, domain.ErrPerfilNaoEncontrado) {
		t.Errorf("BuscarPorID: erro = %v, esperado ErrPerfilNaoEncontrado", err)
	}
}

func TestListar(t *testing.T) {
	repo := storage.NewClientRepository(novoBanco(t))
	ctx := context.Background()

	vazia, err := repo.Listar(ctx)
	if err != nil {
		t.Fatalf("listando banco vazio: %v", err)
	}
	// Precisa ser uma lista vazia, não nula: o JSON deve sair [] e não null.
	if vazia == nil {
		t.Fatal("Listar devolveu nil em vez de lista vazia")
	}
	if len(vazia) != 0 {
		t.Fatalf("banco vazio devolveu %d registros", len(vazia))
	}

	for range 3 {
		if _, err := repo.Criar(ctx, uuid.New(), dadosDeTeste()); err != nil {
			t.Fatalf("criando: %v", err)
		}
	}

	todos, err := repo.Listar(ctx)
	if err != nil {
		t.Fatalf("listando: %v", err)
	}
	if len(todos) != 3 {
		t.Fatalf("listou %d, esperado 3", len(todos))
	}
}

func TestRemover(t *testing.T) {
	repo := storage.NewClientRepository(novoBanco(t))
	ctx := context.Background()
	conta := uuid.New()

	if _, err := repo.Criar(ctx, conta, dadosDeTeste()); err != nil {
		t.Fatalf("criando: %v", err)
	}

	if err := repo.Remover(ctx, conta); err != nil {
		t.Fatalf("removendo: %v", err)
	}

	if _, err := repo.BuscarPorConta(ctx, conta); !errors.Is(err, domain.ErrPerfilNaoEncontrado) {
		t.Errorf("o perfil ainda existe após remover")
	}

	if err := repo.Remover(ctx, conta); !errors.Is(err, domain.ErrPerfilNaoEncontrado) {
		t.Errorf("remover de novo: erro = %v, esperado ErrPerfilNaoEncontrado", err)
	}
}

// Migrar roda em toda inicialização; rodar duas vezes não pode quebrar nem
// apagar dados.
func TestMigrarEIdempotente(t *testing.T) {
	pool := novoBanco(t)
	ctx := context.Background()
	repo := storage.NewClientRepository(pool)

	conta := uuid.New()
	if _, err := repo.Criar(ctx, conta, dadosDeTeste()); err != nil {
		t.Fatalf("criando: %v", err)
	}

	if err := storage.Migrar(ctx, pool); err != nil {
		t.Fatalf("migrando de novo: %v", err)
	}

	if _, err := repo.BuscarPorConta(ctx, conta); err != nil {
		t.Fatalf("o dado sumiu após migrar de novo: %v", err)
	}
}
