// Package auth valida os tokens emitidos pelo Authorization Service.
//
// Este serviço nunca chama o Authorization para autenticar uma requisição: ele
// carrega a chave pública uma vez, na inicialização, e verifica a assinatura
// localmente. A chave pública só permite verificar — não permite emitir tokens.
package auth

import (
	"crypto/rsa"
	"errors"
	"fmt"
	"os"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	ErrTokenInvalido = errors.New("token inválido")
	ErrTokenExpirado = errors.New("token expirado")
)

// Claims são os dados que o Authorization coloca dentro do token.
type Claims struct {
	AccountID uuid.UUID
	Email     string
	Role      string
}

// IsAdmin informa se a conta tem o papel administrativo.
func (c Claims) IsAdmin() bool {
	return c.Role == "ADMIN"
}

// Verifier confere tokens contra a chave pública do Authorization.
type Verifier struct {
	publicKey *rsa.PublicKey
	issuer    string
}

// NewVerifier carrega a chave pública em PEM a partir do caminho informado.
func NewVerifier(publicKeyPath, issuer string) (*Verifier, error) {
	pemBytes, err := os.ReadFile(publicKeyPath)
	if err != nil {
		// Erro comum em clone novo: a pasta keys/ não é versionada, e o Docker
		// cria o diretório vazio ao montar o volume em vez de reclamar. Sem esta
		// dica, a mensagem seria só "arquivo não encontrado".
		return nil, fmt.Errorf(
			"chave pública não encontrada em %s: %w\n"+
				"Rode ./scripts/fetch-public-key.sh com o Authorization Service no ar",
			publicKeyPath, err)
	}

	publicKey, err := jwt.ParseRSAPublicKeyFromPEM(pemBytes)
	if err != nil {
		return nil, fmt.Errorf("interpretando a chave pública: %w", err)
	}

	return &Verifier{publicKey: publicKey, issuer: issuer}, nil
}

// Verify confere assinatura, emissor, tipo e validade do token.
func (v *Verifier) Verify(tokenString string) (*Claims, error) {
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (any, error) {
		// Exigir RS256 explicitamente é o que impede o ataque de troca de
		// algoritmo: sem esta checagem, alguém poderia assinar um token com
		// HMAC usando a chave pública (que é conhecida) como se fosse senha,
		// ou enviar alg "none".
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("algoritmo inesperado: %v", t.Header["alg"])
		}
		return v.publicKey, nil
	},
		jwt.WithIssuer(v.issuer),
		jwt.WithValidMethods([]string{"RS256"}),
		jwt.WithExpirationRequired(),
	)

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpirado
		}
		return nil, ErrTokenInvalido
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, ErrTokenInvalido
	}

	// Um refresh token não pode abrir rota protegida.
	if tipo, _ := claims["typ"].(string); tipo != "access" {
		return nil, ErrTokenInvalido
	}

	subject, _ := claims["sub"].(string)
	accountID, err := uuid.Parse(subject)
	if err != nil {
		return nil, ErrTokenInvalido
	}

	email, _ := claims["email"].(string)
	role, _ := claims["role"].(string)

	return &Claims{AccountID: accountID, Email: email, Role: role}, nil
}
