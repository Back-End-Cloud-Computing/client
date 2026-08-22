// Package storage guarda os perfis no PostgreSQL.
package storage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Back-End-Cloud-Computing/client/internal/domain"
)

type ClientRepository struct {
	pool *pgxpool.Pool
}

func NewClientRepository(pool *pgxpool.Pool) *ClientRepository {
	return &ClientRepository{pool: pool}
}

// Conectar abre o pool e confirma que o banco responde.
func Conectar(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("abrindo conexão com o Postgres: %w", err)
	}

	ctxPing, cancelar := context.WithTimeout(ctx, 10*time.Second)
	defer cancelar()

	if err := pool.Ping(ctxPing); err != nil {
		pool.Close()
		return nil, fmt.Errorf("Postgres não respondeu: %w", err)
	}

	return pool, nil
}

// Migrar cria a tabela, se ainda não existir.
func Migrar(ctx context.Context, pool *pgxpool.Pool) error {
	const ddl = `
CREATE TABLE IF NOT EXISTS perfil_cliente (
    id            UUID PRIMARY KEY,
    -- Conta no Authorization Service. UNIQUE: um perfil por conta.
    account_id    UUID NOT NULL UNIQUE,
    nome          VARCHAR(120) NOT NULL,
    telefone      VARCHAR(20)  NOT NULL,
    logradouro    VARCHAR(150) NOT NULL,
    numero        VARCHAR(20)  NOT NULL,
    complemento   VARCHAR(100),
    bairro        VARCHAR(100),
    cidade        VARCHAR(100) NOT NULL,
    estado        CHAR(2)      NOT NULL,
    cep           VARCHAR(9)   NOT NULL,
    criado_em     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    atualizado_em TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);`

	if _, err := pool.Exec(ctx, ddl); err != nil {
		return fmt.Errorf("criando a tabela perfil_cliente: %w", err)
	}
	return nil
}

const colunas = `id, account_id, nome, telefone, logradouro, numero, complemento,
	bairro, cidade, estado, cep, criado_em, atualizado_em`

func (r *ClientRepository) Criar(ctx context.Context, accountID uuid.UUID,
	dados domain.DadosPerfil) (*domain.Client, error) {

	const sql = `
		INSERT INTO perfil_cliente
			(id, account_id, nome, telefone, logradouro, numero, complemento,
			 bairro, cidade, estado, cep)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		RETURNING ` + colunas

	linha := r.pool.QueryRow(ctx, sql,
		uuid.New(), accountID, dados.Nome, dados.Telefone,
		dados.Endereco.Logradouro, dados.Endereco.Numero, dados.Endereco.Complemento,
		dados.Endereco.Bairro, dados.Endereco.Cidade, dados.Endereco.Estado,
		dados.Endereco.CEP)

	cliente, err := escanear(linha)
	if err != nil {
		// 23505 = violação de UNIQUE: a conta já tem perfil.
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, domain.ErrPerfilJaExiste
		}
		return nil, err
	}
	return cliente, nil
}

func (r *ClientRepository) BuscarPorConta(ctx context.Context, accountID uuid.UUID) (*domain.Client, error) {
	const sql = `SELECT ` + colunas + ` FROM perfil_cliente WHERE account_id = $1`
	return escanear(r.pool.QueryRow(ctx, sql, accountID))
}

func (r *ClientRepository) BuscarPorID(ctx context.Context, id uuid.UUID) (*domain.Client, error) {
	const sql = `SELECT ` + colunas + ` FROM perfil_cliente WHERE id = $1`
	return escanear(r.pool.QueryRow(ctx, sql, id))
}

func (r *ClientRepository) Atualizar(ctx context.Context, accountID uuid.UUID,
	dados domain.DadosPerfil) (*domain.Client, error) {

	const sql = `
		UPDATE perfil_cliente
		SET nome = $2, telefone = $3, logradouro = $4, numero = $5,
		    complemento = $6, bairro = $7, cidade = $8, estado = $9, cep = $10,
		    atualizado_em = NOW()
		WHERE account_id = $1
		RETURNING ` + colunas

	return escanear(r.pool.QueryRow(ctx, sql,
		accountID, dados.Nome, dados.Telefone,
		dados.Endereco.Logradouro, dados.Endereco.Numero, dados.Endereco.Complemento,
		dados.Endereco.Bairro, dados.Endereco.Cidade, dados.Endereco.Estado,
		dados.Endereco.CEP))
}

func (r *ClientRepository) Listar(ctx context.Context) ([]domain.Client, error) {
	const sql = `SELECT ` + colunas + ` FROM perfil_cliente ORDER BY criado_em`

	linhas, err := r.pool.Query(ctx, sql)
	if err != nil {
		return nil, err
	}
	defer linhas.Close()

	clientes := []domain.Client{}
	for linhas.Next() {
		cliente, err := escanear(linhas)
		if err != nil {
			return nil, err
		}
		clientes = append(clientes, *cliente)
	}
	return clientes, linhas.Err()
}

func (r *ClientRepository) Remover(ctx context.Context, accountID uuid.UUID) error {
	etiqueta, err := r.pool.Exec(ctx, `DELETE FROM perfil_cliente WHERE account_id = $1`, accountID)
	if err != nil {
		return err
	}
	if etiqueta.RowsAffected() == 0 {
		return domain.ErrPerfilNaoEncontrado
	}
	return nil
}

// linhaEscaneavel cobre tanto pgx.Row quanto pgx.Rows.
type linhaEscaneavel interface {
	Scan(destinos ...any) error
}

func escanear(linha linhaEscaneavel) (*domain.Client, error) {
	var c domain.Client
	var complemento, bairro *string

	err := linha.Scan(&c.ID, &c.AccountID, &c.Nome, &c.Telefone,
		&c.Endereco.Logradouro, &c.Endereco.Numero, &complemento,
		&bairro, &c.Endereco.Cidade, &c.Endereco.Estado, &c.Endereco.CEP,
		&c.CriadoEm, &c.AtualizadoEm)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrPerfilNaoEncontrado
	}
	if err != nil {
		return nil, err
	}

	if complemento != nil {
		c.Endereco.Complemento = *complemento
	}
	if bairro != nil {
		c.Endereco.Bairro = *bairro
	}

	return &c, nil
}
