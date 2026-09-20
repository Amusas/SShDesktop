# SSH Web Terminal & Virtual Machine Manager (Plantillas y Discos Multiattach)

App web interactiva de terminal SSH en tiempo real y gestor de Máquinas Virtuales basado en **Plantillas preconfiguradas con discos Multiattach en Oracle VirtualBox 7.2**, aprovisionamiento automático mediante usuario bootstrap, generación nativa de claves **Ed25519** por VM, verificación estricta de `known_hosts` y conexión terminal multitarget (Host Windows o VM invitada).

---

## 📋 Requisitos

- [Go 1.22+](https://go.dev/dl/)
- Oracle VirtualBox 7.2 instalado en el host remoto Windows
- Servidor OpenSSH habilitado en el host remoto con acceso mediante llave privada
- Directorio de plantillas en `C:\VMManager\Templates` con sus respectivos archivos `template.json` y `base.vdi`
- Clave privada de bootstrap en `C:\VMManager\SSH\bootstrap_id_ed25519`

---

## 📁 Estructura del Proyecto

```
cmd/ssh-web/          Punto de entrada del servidor (main.go)
internal/api/         Controladores HTTP REST API (/api/templates, /api/vms, /api/hosts, /api/keys, /ws)
internal/template/    Descubrimiento, validación y sanitización de plantillas remotas (C:\VMManager\Templates)
internal/ssh/         Cliente SSH, terminal PTY, VBoxManage 7.2 (Multiattach, NAT forwarding), provisioning y known_hosts
internal/config/      Configuración jerárquica (config.properties, .env y runtime)
internal/vm/          Lógica de ciclo de vida de VMs, asignador de puertos (PortAllocator) y almacén local (vms.json)
internal/keys/        Generador criptográfico nativo de llaves Ed25519 y derivador de usuarios únicos
pkg/logger/           Logger estructurado JSON (slog)
web/                  Frontend interactivo (HTML5, CSS3, JavaScript vanilla)
  css/style.css       Estilos visuales, tema oscuro, animaciones de spinner y botones de acción
  js/app.js           Controlador de UI, selector de plantillas, ciclo de vida de VMs y WebSocket
  js/ansi.js          Decodificador y parseador robusto de secuencias ANSI / VT100
  js/websocket.js     Manejador del ciclo de vida y reconexión de WebSockets
```

---

## ⚙️ Configuración (`config.properties` / `.env`)

| Propiedad (`config.properties`) | Variable (`.env`) | Requerida | Por Defecto | Descripción |
|---------------------------------|-------------------|-----------|-------------|-------------|
| `ssh.host`                      | `SSH_HOST`        | Sí        | -           | IP del host Windows remoto con VirtualBox |
| `ssh.port`                      | `SSH_PORT`        | No        | `22`        | Puerto SSH del host Windows |
| `ssh.user`                      | `SSH_USER`        | Sí        | -           | Usuario del host Windows |
| `ssh.key_path`                  | `SSH_KEY_PATH`    | Sí        | -           | Llave privada para conectar al host Windows |
| `vm.template_root`              | `VM_TEMPLATE_ROOT`| No        | `C:\VMManager\Templates` | Directorio raíz de plantillas base |
| `vm.vm_root`                    | `VM_VM_ROOT`      | No        | `C:\VMManager\VMs`       | Directorio de almacenamiento de VMs |
| `vm.ssh_root`                   | `VM_SSH_ROOT`     | No        | `C:\VMManager\SSH`       | Directorio de identidades SSH y `known_hosts` |
| `vm.bootstrap_key_path`         | `VM_BOOTSTRAP_KEY_PATH` | No  | `C:\VMManager\SSH\bootstrap_id_ed25519` | Clave privada bootstrap del backend |
| `vm.base_port`                  | `VM_BASE_PORT`    | No        | `22000`     | Puerto base para reenvío NAT SSH |
| `server.addr`                   | `SERVER_ADDR`     | No        | `:8080`     | Dirección de escucha del servidor web |
| `log.level`                     | `LOG_LEVEL`       | No        | `info`      | Nivel de logs: `debug`, `info`, `warn`, `error` |

---

## 🚀 Preparación de la Primera Plantilla

Para añadir una plantilla en el host Windows:

1. Crear la carpeta en `C:\VMManager\Templates\<id-plantilla>` (ej. `C:\VMManager\Templates\ubuntu-2404`).
2. Copiar el disco base `base.vdi` previamente preparado (con `openssh-server`, usuario `bootstrap` y clave pública bootstrap en `~/.ssh/authorized_keys`).
3. Crear el archivo `template.json`:
```json
{
  "id": "ubuntu-2404",
  "name": "Ubuntu 24.04 LTS",
  "osType": "Ubuntu_64",
  "disk": "base.vdi",
  "sshUser": "bootstrap",
  "sshPort": 22
}
```

---

## 🌟 Ciclo de Vida y Flujo Automatizado de Creación

1. **Selección de Plantilla**: El formulario web obtiene las plantillas válidas mediante `GET /api/templates`.
2. **Asignación de Puerto**: `PortAllocator` asigna un puerto libre de reenvío NAT (ej. `22001`).
3. **Generación de Identidad**: Se deriva un usuario único Linux (ej. `vm_otramv_a82f`) y un par de claves **Ed25519**.
4. **Comandos VirtualBox 7.2**:
   - `VBoxManage createvm --name "otraMV" --ostype "Ubuntu_64" --register`
   - `VBoxManage modifyvm "otraMV" --memory 4096 --cpus 2 --nic1 nat --natpf1 "ssh,tcp,,22001,,22"`
   - `VBoxManage storagectl "otraMV" --name "SATA" --add sata --controller IntelAhci`
   - `VBoxManage storageattach "otraMV" --storagectl "SATA" --port 0 --device 0 --type hdd --medium "C:\VMManager\Templates\ubuntu-2404\base.vdi" --mtype multiattach`
5. **Aprovisionamiento SSH**:
   - Se inicia la VM en modo headless: `VBoxManage startvm "otraMV" --type headless`.
   - Se espera a que el puerto SSH esté listo (`WaitForSSH`).
   - El backend conecta con la clave `bootstrap` y crea el usuario único `vm_otramv_a82f`, instala su clave pública en `authorized_keys` y desactiva el acceso `bootstrap`.
   - Se captura la **Host Key** de la VM y se registra en `C:\VMManager\SSH\known_hosts`.
6. **Operaciones Disponibles**:
   - **▶ Iniciar**: Arranque headless con control de estado.
   - **⏹ Apagar (ACPI)**: `VBoxManage controlvm "otraMV" acpipowerbutton` con espera a `poweroff`.
   - **⚡ Poweroff**: `VBoxManage controlvm "otraMV" poweroff` forzado inmediato.
   - **>_ Terminal**: Abre sesión SSH interactiva web directamente contra la VM usando su usuario único y clave Ed25519.
   - **🗑 Eliminar**: `VBoxManage unregistervm "otraMV" --delete` sin tocar el `base.vdi` de la plantilla.

---

## 🛠️ Ejecución y Compilación

```powershell
# Compilar el ejecutable binario
go build -o ssh-web.exe ./cmd/ssh-web

# Iniciar la aplicación
.\ssh-web.exe
```

Abre **http://localhost:8080** en tu navegador.
