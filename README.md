# SSH Web Terminal

App web de terminal SSH en tiempo real. Permite conectarse a un servidor remoto via SSH y visualizar/interactuar con la consola directamente desde el navegador, con soporte para PowerShell, CMD y configuración dinámica en caliente.

## Requisitos

- [Go 1.27+](https://go.dev/dl/)
- Acceso SSH a un servidor remoto mediante llave privada

## Estructura del Proyecto

```
cmd/ssh-web/          Punto de entrada (main.go)
internal/api/         Controladores HTTP, REST API y WebSocket
internal/ssh/         Cliente SSH, PTY, soporte de shells (CMD/PowerShell) y flujos I/O
internal/config/      Configuracion via config.properties, .env y variables de entorno
pkg/logger/           Logs estructurados JSON (slog)
web/                  Frontend (HTML, CSS, JS)
```

## Mecanismo de Configuración

La aplicación admite múltiples fuentes de configuración ordenadas por jerarquía:
1. **Archivo de propiedades (`config.properties`)**: Define los valores predeterminados de la aplicación.
2. **Archivo `.env`**: Sobrescribe o complementa los valores de propiedades.
3. **Variables de entorno del sistema**: Tienen la mayor precedencia en el arranque.
4. **Interfaz Web (Menú Superior Izquierdo)**: Permite modificar cualquiera de los parámetros dinámicamente y reconectarse sin reiniciar el servidor.

### Variables de Configuración

| Propiedad (`config.properties`) | Variable (`.env`) | Requerida | Descripción |
|---------------------------------|-------------------|-----------|-------------|
| `ssh.host`                      | `SSH_HOST`        | Sí        | IP o hostname del servidor SSH |
| `ssh.port`                      | `SSH_PORT`        | No (22)   | Puerto del servidor SSH |
| `ssh.user`                      | `SSH_USER`        | Sí        | Usuario SSH |
| `ssh.shell`                     | `SSH_SHELL`       | No (pwsh) | Terminal/Shell: `powershell`, `cmd`, o `default` |
| `ssh.start_path`                | `SSH_START_PATH`  | No        | Ruta predeterminada de inicio |
| `ssh.key_path`                  | `SSH_KEY_PATH`    | Sí        | Ruta a la llave privada SSH |
| `server.addr`                   | `SERVER_ADDR`     | No (:8080)| Dirección del servidor web |
| `log.level`                     | `LOG_LEVEL`       | No (info) | Nivel de log: debug, info, warn, error |

## Ejecución

```powershell
# Instalar dependencias
go mod tidy

# Ejecutar el servidor:
go run ./cmd/ssh-web
```

Abre **http://localhost:8080** en el navegador.

## Funcionalidades Principales

### Sección de Configuración en la WebApp (Esquina Superior Izquierda)
- **Menú de configuración**: Accesible con el engranaje (⚙) en la parte superior izquierda.
- **Campos editables en caliente**:
  - Host / IP y Puerto.
  - Usuario SSH.
  - **Selector de Shell**: Alterna entre **PowerShell** (`powershell.exe`), **CMD** (`cmd.exe`) o **Shell por Defecto**.
  - **Ruta predeterminada de inicio**: Directorio en el que iniciará la consola al conectarse.
  - Ubicación de la llave privada SSH.
- **Persistencia**: Al presionar **"Guardar y Conectar"**, los valores se persisten en `config.properties` y se reconecta la sesión inmediatamente.
- **Valores por defecto**: Botón para restaurar los parámetros iniciales definidos en las variables de entorno.

### Terminal Web
- Emulación ANSI completa (colores 256, negrita, borrado de pantalla `cls`/`clear`, retornos de carro `\r`).
- Historial de comandos en la interfaz con flechas Arriba/Abajo.
- Detección automática del prompt de Windows CMD (`C:\path>`), PowerShell (`PS C:\path>`) y Linux.
- Redimensionamiento reactivo del PTY según el tamaño de la ventana del navegador.

## Build

```powershell
go build -o ssh-web.exe ./cmd/ssh-web
```
