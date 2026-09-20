package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/feliren/ssh-web/internal/api"
	"github.com/feliren/ssh-web/internal/config"
	"github.com/feliren/ssh-web/pkg/logger"
)

func main() {
	// Carga de variables de entorno y validacion fail-fast
	cfg, err := config.Load()
	if err != nil {
		slog.Error("error de configuracion", "error", err)
		os.Exit(1)
	}

	log := logger.New(cfg.LogLevel)
	log.Info("iniciando emulador de terminal ssh-web")

	staticDir := "web"
	if _, err := os.Stat(staticDir); os.IsNotExist(err) {
		log.Error("directorio de estaticos no encontrado", "ruta", staticDir)
		os.Exit(1)
	}

	handler := api.NewHandler(cfg, log, staticDir)

	server := &http.Server{
		Addr:         cfg.ServerAddr,
		Handler:      handler,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 0, // Desactiva el timeout global para soportar aprovisionamiento de VMs (2-4 min) y WebSockets
		IdleTimeout:  120 * time.Second,
	}

	// Inicio asincrono del servidor HTTP
	go func() {
		log.Info("servidor HTTP a la escucha", "direccion", cfg.ServerAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("error en servidor HTTP", "error", err)
			os.Exit(1)
		}
	}()

	// Captura de senales de interrupcion para apagado controlado
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit

	log.Info("senal de apagado recibida", "senal", sig.String())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Error("fallo en el apagado controlado", "error", err)
		os.Exit(1)
	}

	log.Info("servidor detenido correctamente")
}
