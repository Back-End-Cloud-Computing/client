package auth_test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/Back-End-Cloud-Computing/client/internal/auth"
)

const emissor = "ganjj-authorization"

// chaveDeTeste cria um par RSA e grava a chave pública em PEM, do mesmo jeito
// que o Authorization Service publica em GET /auth/public-key.
func chaveDeTeste(t *testing.T) (*rsa.PrivateKey, string) {
	t.Helper()

	chave, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("gerando chave: %v", err)
	}

	der, err := x509.MarshalPKIXPublicKey(&chave.PublicKey)
	if err != nil {
		t.Fatalf("serializando chave pública: %v", err)
	}

	caminho := filepath.Join(t.TempDir(), "public.pem")
	arquivo := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})
	if err := os.WriteFile(caminho, arquivo, 0o600); err != nil {
		t.Fatalf("gravando PEM: %v", err)
	}

	return chave, caminho
}

// tokenComo monta um token igual ao que o serviço Java emite.
func tokenComo(t *testing.T, chave *rsa.PrivateKey, claims jwt.MapClaims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	assinado, err := token.SignedString(chave)
	if err != nil {
		t.Fatalf("assinando token: %v", err)
	}
	return assinado
}

func claimsDeAcesso(conta uuid.UUID) jwt.MapClaims {
	agora := time.Now()
	return jwt.MapClaims{
		"iss":   emissor,
		"sub":   conta.String(),
		"email": "cliente@ganjj.com",
		"role":  "CLIENTE",
		"typ":   "access",
		"iat":   agora.Unix(),
		"exp":   agora.Add(15 * time.Minute).Unix(),
	}
}

func TestAceitaTokenLegitimo(t *testing.T) {
	chave, caminho := chaveDeTeste(t)
	verificador, err := auth.NewVerifier(caminho, emissor)
	if err != nil {
		t.Fatalf("criando verificador: %v", err)
	}

	conta := uuid.New()
	claims, err := verificador.Verify(tokenComo(t, chave, claimsDeAcesso(conta)))
	if err != nil {
		t.Fatalf("token legítimo foi recusado: %v", err)
	}

	if claims.AccountID != conta {
		t.Errorf("conta = %v, esperado %v", claims.AccountID, conta)
	}
	if claims.Email != "cliente@ganjj.com" {
		t.Errorf("email = %q", claims.Email)
	}
	if claims.IsAdmin() {
		t.Error("conta CLIENTE não deveria ser admin")
	}
}

func TestReconheceAdmin(t *testing.T) {
	chave, caminho := chaveDeTeste(t)
	verificador, _ := auth.NewVerifier(caminho, emissor)

	claims := claimsDeAcesso(uuid.New())
	claims["role"] = "ADMIN"

	resultado, err := verificador.Verify(tokenComo(t, chave, claims))
	if err != nil {
		t.Fatalf("token de admin recusado: %v", err)
	}
	if !resultado.IsAdmin() {
		t.Error("papel ADMIN não foi reconhecido")
	}
}

// A defesa mais importante deste pacote: a chave pública é conhecida por todos
// os serviços. Se o verificador aceitasse HMAC, qualquer um poderia usar essa
// chave como se fosse a senha e forjar tokens.
func TestRecusaTrocaDeAlgoritmoParaHMAC(t *testing.T) {
	chave, caminho := chaveDeTeste(t)
	verificador, _ := auth.NewVerifier(caminho, emissor)

	publicaDER, _ := x509.MarshalPKIXPublicKey(&chave.PublicKey)
	forjado := jwt.NewWithClaims(jwt.SigningMethodHS256, claimsDeAcesso(uuid.New()))
	assinado, err := forjado.SignedString(publicaDER)
	if err != nil {
		t.Fatalf("montando token forjado: %v", err)
	}

	if _, err := verificador.Verify(assinado); err == nil {
		t.Fatal("FALHA DE SEGURANÇA: token assinado com HMAC foi aceito")
	}
}

func TestRecusaAlgoritmoNone(t *testing.T) {
	_, caminho := chaveDeTeste(t)
	verificador, _ := auth.NewVerifier(caminho, emissor)

	semAssinatura := jwt.NewWithClaims(jwt.SigningMethodNone, claimsDeAcesso(uuid.New()))
	assinado, err := semAssinatura.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("montando token sem assinatura: %v", err)
	}

	if _, err := verificador.Verify(assinado); err == nil {
		t.Fatal("FALHA DE SEGURANÇA: token com alg=none foi aceito")
	}
}

func TestRecusaTokenDeOutraChave(t *testing.T) {
	_, caminho := chaveDeTeste(t)
	verificador, _ := auth.NewVerifier(caminho, emissor)

	outraChave, _ := rsa.GenerateKey(rand.Reader, 2048)

	if _, err := verificador.Verify(tokenComo(t, outraChave, claimsDeAcesso(uuid.New()))); err == nil {
		t.Fatal("token assinado por outra chave foi aceito")
	}
}

func TestRecusaRefreshTokenComoAcesso(t *testing.T) {
	chave, caminho := chaveDeTeste(t)
	verificador, _ := auth.NewVerifier(caminho, emissor)

	claims := claimsDeAcesso(uuid.New())
	claims["typ"] = "refresh"

	if _, err := verificador.Verify(tokenComo(t, chave, claims)); err == nil {
		t.Fatal("refresh token foi aceito como token de acesso")
	}
}

func TestRecusaTokenExpirado(t *testing.T) {
	chave, caminho := chaveDeTeste(t)
	verificador, _ := auth.NewVerifier(caminho, emissor)

	claims := claimsDeAcesso(uuid.New())
	claims["exp"] = time.Now().Add(-1 * time.Minute).Unix()

	_, err := verificador.Verify(tokenComo(t, chave, claims))
	if err != auth.ErrTokenExpirado {
		t.Fatalf("erro = %v, esperado ErrTokenExpirado", err)
	}
}

func TestRecusaEmissorDesconhecido(t *testing.T) {
	chave, caminho := chaveDeTeste(t)
	verificador, _ := auth.NewVerifier(caminho, emissor)

	claims := claimsDeAcesso(uuid.New())
	claims["iss"] = "outro-emissor"

	if _, err := verificador.Verify(tokenComo(t, chave, claims)); err == nil {
		t.Fatal("token de outro emissor foi aceito")
	}
}

func TestRecusaTokenSemExpiracao(t *testing.T) {
	chave, caminho := chaveDeTeste(t)
	verificador, _ := auth.NewVerifier(caminho, emissor)

	claims := claimsDeAcesso(uuid.New())
	delete(claims, "exp")

	if _, err := verificador.Verify(tokenComo(t, chave, claims)); err == nil {
		t.Fatal("token sem expiração foi aceito")
	}
}

func TestFalhaSeChavePublicaNaoExiste(t *testing.T) {
	if _, err := auth.NewVerifier("/caminho/que/nao/existe.pem", emissor); err == nil {
		t.Fatal("deveria falhar ao não encontrar a chave pública")
	}
}
