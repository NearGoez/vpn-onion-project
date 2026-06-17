# 🚀 Guía de Desarrollo: Capturador de Red de Bajo Nivel (vpn-bridge)

Este documento detalla el plan de ejecución y la arquitectura técnica para el componente `vpn-bridge` desarrollado en C++. La función crítica de este módulo es actuar como el puente exclusivo entre el tráfico de red de la capa de kernel y el entorno criptográfico controlado en el espacio de usuario.

---

## 📂 1. Arquitectura de Código y Árbol de Directorios

El desarrollo se organiza bajo una estructura modular para facilitar la compilación limpia y el aislamiento de responsabilidades:

```text
vpn-bridge/
├── Makefile                 # Reglas de compilación y optimización estricta.
├── include/                 # Declaraciones de interfaces y cabeceras (.hpp).
│   ├── event_loop.hpp
│   ├── ipc_socket.hpp
│   └── tun_device.hpp
├── src/                     # Código de implementación (.cpp).
│   ├── main.cpp             # Punto de entrada y manejo de señales.
│   ├── event_loop.cpp
│   ├── ipc_socket.cpp
│   └── tun_device.cpp
└── tests/
    └── mock_receiver.py     # Script de emulación para pruebas de caja negra.

```

---

## 🏗️ 2. Plan de Trabajo por Hitos

El cronograma está estructurado de forma secuencial, garantizando la estabilidad de las llamadas al sistema antes de acoplar la lógica de comunicación con el binario de Go.

### 🔌 Hito 1: Controladores y Setup de Interfaz TUN

* **Tiempo Estimado:** 6 horas.
* **Objetivo:** Registrar una interfaz de red virtual en el kernel de Linux y obtener un descriptor de archivo (FD) válido para leer bytes IP nativos.
* **Archivos Involucrados:**
* `src/vpn-bridge/include/tun_device.hpp`
* `src/vpn-bridge/src/tun_device.cpp`


* **Mecanismo:**
* Apertura del clonador de red mediante `open("/dev/net/tun", O_RDWR)`.
* Configuración de la estructura del sistema `struct ifreq` asignando los flags `IFF_TUN | IFF_NO_PI` (Modo túnel sin cabeceras de paquete adicionales).
* Ejecución de `ioctl(fd, TUNSETIFF, (void *) &ifr)` para levantar la interfaz virtual (ej. `vpn0`).


* **Relación con el Contrato IPC:** Establece la restricción física del buffer; al configurar la interfaz en modo `IFF_TUN`, el kernel garantiza que los bytes leídos inician estrictamente con el byte de versión de la cabecera IP (Capa 3), cumpliendo el contrato de payloads puros acordado con Go.

### 🕳️ Hito 2: Cliente del Canal Local IPC

* **Tiempo Estimado:** 6 hours.
* **Objetivo:** Conectarse al socket Unix creado por el motor criptográfico e implementar la transferencia atómica de datagramas.
* **Archivos Involucrados:**
* `src/vpn-bridge/include/ipc_socket.hpp`
* `src/vpn-bridge/src/ipc_socket.cpp`


* **Mecanismo:**
* Instanciación del socket local con `socket(AF_UNIX, SOCK_DGRAM, 0)`.
* Asignación de la estructura de dirección `struct sockaddr_un` apuntando a la ruta inmutable definida en el contrato.
* Implementación de la llamada `connect()` hacia el path del socket.


* **Relación con el Contrato IPC:** El código asume de forma estricta que la ruta del socket es `/tmp/onion_vpn.sock` y que opera bajo la semántica de `SOCK_DGRAM`, enviando y recibiendo paquetes completos por cada llamada del sistema, sin divisores de flujo.

### 🔄 Hito 3: Event Loop Asíncrono Bilateral

* **Tiempo Estimado:** 8 horas.
* **Objetivo:** Orquestar el flujo continuo de bytes entre la interfaz TUN y el socket IPC de manera no bloqueante.
* **Archivos Involucrados:**
* `src/vpn-bridge/include/event_loop.hpp`
* `src/vpn-bridge/src/event_loop.cpp`


* **Mecanismo:**
* Configuración de una estructura `struct pollfd fds[2]` para monitorear simultáneamente el descriptor del TUN y el descriptor del socket UNIX.
* **Pipeline de Subida:** Si hay datos listos en `tun_fd` $\rightarrow$ `read()` de longitud fija (MTU 1500) y retransmisión directa con `send()` al socket UNIX.
* **Pipeline de Bajada:** Si hay datos listos en `socket_fd` $\rightarrow$ `recv()` del datagramas descifrado de Go y escritura directa con `write()` en `tun_fd`.


* **Relación con el Contrato IPC:** Mantiene el tamaño del buffer estático en un límite estricto de 1500-2048 bytes para evitar problemas de truncado o fragmentación en el transporte local.

### 🛡️ Hito 4: Gestión de Señales y Hardening del Sistema

* **Tiempo Estimado:** 5 horas.
* **Objetivo:** Evitar fugas de recursos en el kernel y estados zombi en la red ante cancelaciones abruptas.
* **Archivos Involucrados:**
* `src/vpn-bridge/src/main.cpp`
* `src/vpn-bridge/Makefile`


* **Mecanismo:**
* Intercepción de señales mediante `sigaction()` para capturar de forma segura `SIGINT` (`Ctrl+C`) y `SIGTERM`.
* Implementación de una subrutina de cierre donde se ejecute de manera explícita la liberación de los descriptores de archivo con `close(tun_fd)` y `close(socket_fd)`.
* Configuración del `Makefile` con directivas de optimización agresivas para minimizar la latencia de copia de memoria (`-O3 -Wall -Wextra -std=c++17`).


* **Relación con el Contrato IPC:** Al cerrar limpiamente el cliente descriptor de C++, el socket del lado del servidor de Go detecta la desconexión del canal sin dejar buffers corruptos ni descriptores huérfanos en el sistema de archivos local.

---

## 🧪 3. Estrategia de Pruebas Aisladas

Para verificar el correcto funcionamiento de `vpn-bridge` sin depender de la ejecución del código en Go, se utiliza el script provisto en `tests/mock_receiver.py`.

Este script levanta de forma manual el socket de datagramas UNIX en `/tmp/onion_vpn.sock` y dumpea en la consola en formato hexadecimal cualquier paquete IP crudo extraído del kernel por el binario de C++, validando la integridad estructural de la cabecera de Capa 3 de forma totalmente autónoma.
