
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


```

+-----------------------------------------------------------+

| Paquete IP Crudo |

| (Tamaño variable: de 20 bytes hasta el MTU configurado) |

+-----------------------------------------------------------+

^

|

Primer byte (Cabecera IP)

```

### Reglas Estrictas del Buffer:
1. **Sin Metadatos Propios:** No se añadirán prefijos, IDs, delimitadores, ni cabeceras personalizadas por parte de C++ o Go en este canal.
2. **Estructura del Primer Byte:** El byte inicial del buffer siempre corresponde al inicio de la cabecera IP estándar.
   * Si los primeros 4 bits del byte son `0100` (`0x4`), es un paquete **IPv4**. El paquete comienza con el byte `0x45` o similar.
   * Si los primeros 4 bits del byte son `0110` (`0x6`), es un paquete **IPv6**.
3. **Unidad Máxima de Transmisión (MTU):** El tamaño máximo de los datagramas se fija en **1500 bytes**. Los buffers de lectura/escritura en ambos programas deben inicializarse estáticamente con un tamaño de `1500` bytes (o `2048` para margen de seguridad) para evitar truncado de datos.

---

## 3. Direccionalidad del Flujo y Comportamiento esperado

### Flujo A: Subida (`vpn-bridge` -> `vpn-core`)
* **Origen (C++):** Lee del descriptor `/dev/net/tun` un paquete IP generado por el sistema operativo.
* **Acción (C++):** Llama a `send()` / `sendto()` metiendo los bytes exactos en `/tmp/onion_vpn.sock`.
* **Destino (Go):** Recibe el datagrama mediante `ReadFromUnix` o similar, procesa el buffer directamente como un paquete de capa 3 y arranca el pipeline criptográfico.

### Flujo B: Bajada (`vpn-core` -> `vpn-bridge`)
* **Origen (Go):** Tras descifrar y desenvolver las capas onion de la red UDP, obtiene el paquete IP original del destino.
* **Acción (Go):** Escribe los bytes puros del paquete descifrado en `/tmp/onion_vpn.sock` usando `WriteTo`.
* **Destino (C++):** El bucle (`poll`/`select`) detecta actividad en el socket local, lee el datagrama con `recv()` y hace un `write()` directo en el descriptor del TUN para reinyectarlo al kernel de Linux.

---

## 4. Pruebas de Cumplimiento de Contrato (Mocking)

Cualquiera de los dos componentes puede testearse de forma aislada respetando estas firmas:

* **Para probar Go sin C++:** Ejecutar `socat - UNIX-SENDTO:/tmp/onion_vpn.sock` e inyectar bytes hexadecimales dummy para simular la llegada de un paquete IP.
* **Para probar C++ sin Go:** Crear un script corto en Python o usar `socat UNIX-RECVFROM:/tmp/onion_vpn.sock,fork -` para dumpear el tráfico que C++ está extrayendo del dispositivo TUN y verificar que mantenga la estructura IP nativa.


