// Package domain descreve o perfil de cliente do GANJJ.
package domain

import (
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	ErrPerfilNaoEncontrado = errors.New("perfil não encontrado")
	ErrPerfilJaExiste      = errors.New("esta conta já possui um perfil")
)

// Endereco de entrega do cliente.
type Endereco struct {
	Logradouro  string `json:"logradouro"`
	Numero      string `json:"numero"`
	Complemento string `json:"complemento,omitempty"`
	Bairro      string `json:"bairro"`
	Cidade      string `json:"cidade"`
	Estado      string `json:"estado"`
	CEP         string `json:"cep"`
}

// Client é o perfil de um cliente.
//
// Não guarda e-mail nem senha: credenciais são do Authorization Service. O elo
// entre os dois é o AccountID, que vem da claim "sub" do token.
type Client struct {
	ID           uuid.UUID `json:"id"`
	AccountID    uuid.UUID `json:"accountId"`
	Nome         string    `json:"nome"`
	Telefone     string    `json:"telefone"`
	Endereco     Endereco  `json:"endereco"`
	CriadoEm     time.Time `json:"criadoEm"`
	AtualizadoEm time.Time `json:"atualizadoEm"`
}

// DadosPerfil é o que o cliente envia ao criar ou atualizar seu perfil.
type DadosPerfil struct {
	Nome     string   `json:"nome"`
	Telefone string   `json:"telefone"`
	Endereco Endereco `json:"endereco"`
}

var (
	regexCEP      = regexp.MustCompile(`^\d{5}-?\d{3}$`)
	regexTelefone = regexp.MustCompile(`^\(?\d{2}\)?\s?9?\d{4}-?\d{4}$`)
)

// Validar confere os campos e devolve os erros por campo, no mesmo formato que
// o Authorization Service usa.
func (d DadosPerfil) Validar() map[string]string {
	erros := map[string]string{}

	if len(strings.TrimSpace(d.Nome)) < 3 {
		erros["nome"] = "O nome deve ter ao menos 3 caracteres."
	}
	if len(d.Nome) > 120 {
		erros["nome"] = "O nome deve ter no máximo 120 caracteres."
	}
	if !regexTelefone.MatchString(strings.TrimSpace(d.Telefone)) {
		erros["telefone"] = "Informe um telefone válido, com DDD."
	}
	if strings.TrimSpace(d.Endereco.Logradouro) == "" {
		erros["endereco.logradouro"] = "O logradouro é obrigatório."
	}
	if strings.TrimSpace(d.Endereco.Numero) == "" {
		erros["endereco.numero"] = "O número é obrigatório."
	}
	if strings.TrimSpace(d.Endereco.Cidade) == "" {
		erros["endereco.cidade"] = "A cidade é obrigatória."
	}
	if len(strings.TrimSpace(d.Endereco.Estado)) != 2 {
		erros["endereco.estado"] = "Use a sigla do estado, com 2 letras."
	}
	if !regexCEP.MatchString(strings.TrimSpace(d.Endereco.CEP)) {
		erros["endereco.cep"] = "Informe um CEP válido (00000-000)."
	}

	return erros
}

// Normalizar limpa espaços e padroniza a sigla do estado.
func (d *DadosPerfil) Normalizar() {
	d.Nome = strings.TrimSpace(d.Nome)
	d.Telefone = strings.TrimSpace(d.Telefone)
	d.Endereco.Logradouro = strings.TrimSpace(d.Endereco.Logradouro)
	d.Endereco.Numero = strings.TrimSpace(d.Endereco.Numero)
	d.Endereco.Complemento = strings.TrimSpace(d.Endereco.Complemento)
	d.Endereco.Bairro = strings.TrimSpace(d.Endereco.Bairro)
	d.Endereco.Cidade = strings.TrimSpace(d.Endereco.Cidade)
	d.Endereco.Estado = strings.ToUpper(strings.TrimSpace(d.Endereco.Estado))
	d.Endereco.CEP = strings.TrimSpace(d.Endereco.CEP)
}
