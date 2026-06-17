# 📄 Contrato de Interfaz IPC: vpn-bridge (C++) ↔ vpn-core (Go)

Este documento establece la especificación técnica estricta e inmutable para la comunicación local entre el capturador de tráfico en C++ (`vpn-bridge`) y el motor criptográfico en Go (`vpn-core`). Ambos componentes deben ceñirse a este formato para garantizar el desarrollo en paralelo sin dependencias de código.

---

## 1. Parámetros del Canal de Comunicación

La comunicación se realizará mediante un **Unix Domain Socket** local optimizado para datagramas.

* **Tipo de Socket:** `SOCK_DGRAM` (En Go: `unixgram`). No se utilizará `SOCK_STREAM` para evitar el overhead de control de flujo y la necesidad de parsear un stream continuo de bytes.
* **Ruta del Archivo Socket:** `/tmp/onion_vpn.sock`
* **Ciclo de Vida del Archivo:** * `vpn-core` (Go) actúa como el **servidor** del socket; es responsable de crear el socket en la ruta asignada, escuchar las conexiones y limpiar el archivo al finalizar (vía `unlink` o `os.Remove`).
  * `vpn-bridge` (C++) actúa como el **cliente**; asume que el socket ya existe en el sistema, se conecta a él y envía/recibe datagramas.

---

## 2. Formato del Mensaje (Payload)

Cada datagrama enviado o recibido a través del socket equivale **exactamente a un (1) paquete IP crudo** obtenido directamente de la interfaz virtual TUN.
+-----------------------------------------------------------+
|                    Paquete IP Crudo                       |
|  (Tamaño variable: de 20 bytes hasta el MTU configurado)  |
+-----------------------------------------------------------+
^
|
Primer byte (Cabecera IP)
