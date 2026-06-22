# 🔌 Guía de Integración y Prueba Real: C++ ↔ Go

Esta guía explica paso a paso cómo probar la integración real del capturador de red en C++ (`vpn-bridge`) y el motor criptográfico en Go (`vpn-core`) en un sistema **Linux** (máquina física, virtual o WSL2 con soporte de red).

---

## 🛠️ Paso 1: Compilar el proyecto completo
En la carpeta raíz del proyecto, ejecuta el siguiente comando para compilar de forma automatizada ambos binarios:
```bash
make build
```
Esto creará:
* El binario de C++ en: `src/vpn-bridge/vpn-bridge`
* El binario de Go en: `src/vpn-core/vpn-core`

---

## 🚀 Paso 2: Ejecutar el Servidor Go (`vpn-core`)
Abre tu **Terminal 1** y ejecuta el cerebro de la VPN en modo simulador (este modo levantará el cliente y los 3 nodos UDP intermedios localmente):
```bash
./src/vpn-core/vpn-core -mode all
```
* **Qué verás:** Mensajes indicando que Go está escuchando en el socket `/tmp/onion_vpn.sock` y abriendo los puertos UDP del 9000 al 9003.

---

## 📡 Paso 3: Ejecutar el Capturador C++ (`vpn-bridge`)
Abre tu **Terminal 2** y ejecuta el programa de Denis con permisos de superusuario (necesario para crear tarjetas de red virtuales en Linux):
```bash
sudo ./src/vpn-bridge/vpn-bridge
```
* **Qué verás:** Mensajes indicando que la interfaz `vpn0` se creó con éxito y que se conectó correctamente al socket `/tmp/onion_vpn.sock` de Go.

---

## ⚙️ Paso 4: Configurar la Tarjeta de Red Virtual `vpn0`
Para que Linux sepa qué IP tiene la interfaz `vpn0` y la active, abre una **Terminal 3** y ejecuta estos dos comandos:
```bash
# Asignamos la dirección IP 10.8.0.1 a la tarjeta virtual vpn0
sudo ip addr add 10.8.0.1/24 dev vpn0

# Levantamos la interfaz para que empiece a recibir tráfico
sudo ip link set vpn0 up
```

---

## 🧪 Paso 5: La Prueba de Fuego (Ping)
En la misma **Terminal 3**, envía paquetes de prueba (pings) a través de la interfaz de la VPN:
```bash
ping -I vpn0 -c 4 10.8.0.5
```

### 📊 Qué debería suceder:
1. El comando `ping` envía un paquete ICMP Request a `10.8.0.5` a través de la tarjeta `vpn0`.
2. El programa C++ de Denis captura el paquete y lo inyecta por el socket a Go.
3. Tu programa en Go lo recibe, le añade las **3 capas de encriptación Onion** y lo envía por UDP al Nodo 1.
4. El paquete pasa cifrado por el Nodo 1 $\rightarrow$ Nodo 2 $\rightarrow$ Nodo 3 (Exit Node).
5. El **Nodo 3 descifra la última capa**, ve que es un ping, **intercambia las IPs de origen y destino** convirtiendo el Request en Reply, y lo vuelve a encriptar de bajada.
6. El paquete viaja de regreso: Nodo 3 $\rightarrow$ Nodo 2 $\rightarrow$ Nodo 1 $\rightarrow$ Cliente Go.
7. Go descifra las 3 capas finales y le devuelve el paquete limpio a Denis por el socket.
8. Denis lo escribe en `vpn0` y Linux recibe la respuesta.

### 🖥️ Resultado esperado en la terminal:
Verás que el ping responde de forma exitosa en la **Terminal 3**:
```text
PING 10.8.0.5 (10.8.0.5) from 10.8.0.1 vpn0: 56(84) bytes of data.
64 bytes from 10.8.0.5: icmp_seq=1 ttl=64 time=1.45 ms
64 bytes from 10.8.0.5: icmp_seq=2 ttl=64 time=1.52 ms
64 bytes from 10.8.0.5: icmp_seq=3 ttl=64 time=1.38 ms
64 bytes from 10.8.0.5: icmp_seq=4 ttl=64 time=1.41 ms

--- 10.8.0.5 ping statistics ---
4 packets transmitted, 4 received, 0% packet loss, time 3004ms
rtt min/avg/max/mdev = 1.380/1.440/1.520/0.052 ms
```

Mientras el ping corre, verás pasar los logs en cascada en las terminales de Go y C++ mostrando cómo se encriptan y desencriptan las celdas en tiempo real.
