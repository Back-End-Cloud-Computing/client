// Command api sobe o Client Service do GANJJ.
//
//	@title			GANJJ — Client Service
//	@version		1.0.0
//	@description	Perfil de cliente do e-commerce GANJJ (nome, telefone, endereço).
//	@description	Credenciais não ficam aqui: e-mail e senha são do Authorization Service.
//	@description
//	@description	Para usar as rotas protegidas, faça login no Authorization
//	@description	(POST http://localhost:8081/auth/login), copie o accessToken e informe
//	@description	em Authorize no formato: Bearer {token}
//
//	@host		localhost:8082
//	@BasePath	/
//
//	@securityDefinitions.apikey	BearerAuth
//	@in							header
//	@name						Authorization
//	@description				Token JWT emitido pelo Authorization Service.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Back-End-Cloud-Computing/client/internal/auth"
	"github.com/Back-End-Cloud-Computing/client/internal/config"
	"github.com/Back-End-Cloud-Computing/client/internal/httpapi"
	"github.com/Back-End-Cloud-Computing/client/internal/storage"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	if err := executar(log); err != nil {
		log.Error("o serviço não pôde iniciar", "erro", err)
		os.Exit(1)
	}
}

func executar(log *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx := context.Background()

	// A chave pública é carregada uma única vez, aqui. A partir deste ponto o
	// serviço valida tokens sozinho, sem falar com o Authorization Service.
	verificador, err := auth.NewVerifier(cfg.JWTPublicKeyPath, cfg.JWTIssuer)
	if err != nil {
		return err
	}
	log.Info("chave pública carregada", "caminho", cfg.JWTPublicKeyPath)

	pool, err := storage.Conectar(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	if err := storage.Migrar(ctx, pool); err != nil {
		return err
	}

	repo := storage.NewClientRepository(pool)
	handler := httpapi.NewHandler(repo, log)

	servidor := &http.Server{
		Addr:              ":" + cfg.ServerPort,
		Handler:           httpapi.Router(handler, verificador, cfg.AllowedOrigins),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// Encerramento ordenado: ao receber SIGTERM (o que o Docker envia ao
	// parar o container), o servidor termina as requisições em andamento.
	encerrar := make(chan os.Signal, 1)
	signal.Notify(encerrar, os.Interrupt, syscall.SIGTERM)

	erroServidor := make(chan error, 1)
	go func() {
		log.Info("client service no ar", "porta", cfg.ServerPort)
		if err := servidor.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			erroServidor <- err
		}
	}()

	select {
	case err := <-erroServidor:
		return err
	case <-encerrar:
		log.Info("encerrando o serviço")
		ctxEncerramento, cancelar := context.WithTimeout(ctx, 15*time.Second)
		defer cancelar()
		return servidor.Shutdown(ctxEncerramento)
	}
}
