# SSH Web Terminal & Virtual Machine Manager

App web interactiva de terminal SSH en tiempo real y gestor de Máquinas Virtuales remotas y Llaves SSH. Permite conectarse a servidores remotos vía SSH, interactuar con la consola (PowerShell, CMD, Bash) directamente desde el navegador, aprovisionar y administrar máquinas virtuales con filtrado dinámico de ISOs y gestionar pares de llaves criptográficas.

---

## 📋 Requisitos

- [Go 1.22+](https://go.dev/dl/)
- Servidor remoto con servicio OpenSSH habilitado y acceso mediante llave privada

---

## 📁 Estructura del Proyecto

```
cmd/ssh-web/          Punto de entrada (main.go)
internal/api/         Controladores HTTP REST API y WebSocket (/ws, /api/config, /api/vms, /api/keys, /api/isos)
internal/ssh/         Cliente SSH, emulación PTY, gestión de shells (CMD / PowerShell / Linux) y flujos I/O
internal/config/      Configuración jerárquica vía config.properties, .env y variables de entorno
internal/vm/          Lógica y almacenamiento persistente de Máquinas Virtuales (vms.json)
internal/keys/        Generador criptográfico y almacén de Llaves SSH RSA 2048 (keys.json, PEM, PUB)
pkg/logger/           Logs estructurados JSON (slog)
web/                  Frontend interactivo (HTML5, CSS3, JavaScript vanilla)
  css/style.css       Estilos visuales, tema oscuro, animaciones y modales
  js/app.js           Controlador de la interfaz, WebSocket, vistas del sidebar y modales
  js/ansi.js          Decodificador y parseador robusto de secuencias ANSI / VT100
  js/websocket.js     Manejador del ciclo de vida y reconexión de WebSockets
```

---

## ⚙️ Mecanismo de Configuración

La aplicación admite múltiples fuentes de configuración ordenadas por precedencia:
1. **Interfaz Web (Menú Lateral > Configuración)**: Permite modificar parámetros dinámicamente y reconectarse en caliente.
2. **Variables de entorno del sistema**: Precedencia en el arranque del servicio.
3. **Archivo `.env`**: Sobrescribe o complementa los valores locales.
4. **Archivo de propiedades (`config.properties`)**: Define los valores predeterminados.

### Variables de Configuración

| Propiedad (`config.properties`) | Variable (`.env`) | Requerida | Por Defecto | Descripción |
|---------------------------------|-------------------|-----------|-------------|-------------|
| `ssh.host`                      | `SSH_HOST`        | Sí        | -           | IP o hostname del servidor SSH |
| `ssh.port`                      | `SSH_PORT`        | No        | `22`        | Puerto del servidor SSH |
| `ssh.user`                      | `SSH_USER`        | Sí        | -           | Usuario SSH |
| `ssh.shell`                     | `SSH_SHELL`       | No        | `pwsh`      | Shell objetivo: `powershell`, `cmd` o `default` |
| `ssh.start_path`                | `SSH_START_PATH`  | No        | `~`         | Ruta inicial predeterminada |
| `ssh.key_path`                  | `SSH_KEY_PATH`    | Sí        | -           | Ruta a la llave privada SSH |
| `server.addr`                   | `SERVER_ADDR`     | No        | `:8080`     | Dirección de escucha del servidor web |
| `log.level`                     | `LOG_LEVEL`       | No        | `info`      | Nivel de logs: `debug`, `info`, `warn`, `error` |

---

## 🚀 Instalación y Ejecución

```powershell
# 1. Descargar dependencias
go mod tidy

# 2. Iniciar el servidor
go run ./cmd/ssh-web

# 3. Compilar el ejecutable binario
go build -o ssh-web.exe ./cmd/ssh-web
```

Abre **http://localhost:8080** en tu navegador.

---

## 🌟 Características y Funcionalidades

### 1. Menú Lateral de Navegación (☰ Hamburguesa)
- Icono de menú de 3 líneas en la esquina superior izquierda.
- Despliega un panel lateral deslizante con 3 módulos principales:
  - **Configuración**: Edición en caliente de host, puerto, usuario, tipo de shell (`PowerShell` / `CMD` / `Default`), rutas y llave SSH.
  - **Máquinas Virtuales**: Lista, creación y gestión de MVs.
  - **Llaves SSH**: Catálogo, descarga y eliminación de claves criptográficas.

---

### 2. Gestión de Máquinas Virtuales
- **Creación de MVs**:
  - Parámetros configurables: Nombre, Sistema Operativo, Imagen ISO, Cores de CPU, Memoria RAM (MB) y Disco (GB).
  - **Filtrado dinámico de ISOs**: Al seleccionar el Sistema Operativo (Ubuntu, Debian, Windows, CentOS, Alpine, Arch), el selector de ISOs se filtra en tiempo real mostrando únicamente las imágenes correspondientes a ese SO.
  - **Aprovisionamiento remoto vía SSH**: Envía los comandos de registro y creación al servidor remoto.
  - **Generación automática de Llave SSH**: Al crear cada máquina virtual, se genera automáticamente un par de llaves SSH asociado a dicha máquina.
- **Visualización de Detalles ("Ver")**:
  - Modal interactivo con las especificaciones completas de la MV: CPU, Memoria RAM formateada, Almacenamiento, SO, ISO, estado, llave SSH asignada y ruta de disco virtual.
- **Eliminación Segura**:
  - Ventana modal de confirmación con el nombre de la máquina resaltado.
  - Animación de spinner de carga (`spin loading`) y bloqueo de botones mientras se procesa la eliminación.

---

### 3. Módulo de Llaves SSH
- **Generación criptográfica**: Claves RSA de 2048 bits generadas en Go, con formato PEM (clave privada) y OpenSSH (clave pública) y cálculo de fingerprint SHA-256.
- **Listado y Asociación**: Muestra a qué máquina virtual pertenece cada llave mediante un badge visual (`🖥️ Nombre_MV`).
- **Descargas Directas**:
  - `⬇ Privada (.pem)`: Descarga el archivo de clave privada para autenticación SSH.
  - `⬇ Pública (.pub)`: Descarga el archivo de clave pública.
- **Eliminación**: Modal de confirmación para revocar y eliminar llaves del almacén.

---

### 4. Terminal Web Interactiva
- **Alineación de Comandos en Línea**: Los comandos ejecutados aparecen inmediatamente a la derecha del carácter del prompt `>` (ej. `hp 245 ryzen 3@HP245G7 C:\Users\andres.HP245G7> dir`) sin saltos de línea intermedios.
- **Resaltado de Comandos en Amarillo**: Todos los comandos ejecutados y el campo de entrada de texto se visualizan en color amarillo (`var(--accent-yellow) = #f59e0b` / `#facc15`), distinguiéndose claramente de la salida del comando.
- **Deduplicación Inteligente de Ecos**: Filtro en tiempo real que previene duplicados repetidos originados por Windows ConPTY y OpenSSH.
- **Limpieza de Pantalla Instantánea**: Soporte completo para `cls` y `clear` que limpia toda la pantalla en una sola ejecución.
- **Soporte ANSI / VT100**: Colores de 256 paletas, negrita, borrado de línea (`\x1b[2K`, `\x1b[1G`), eliminación de artefactos en PowerShell (`PPdir`, `>>P>>`) y auto-scroll inteligente.

