# 🎤 Guía de Presentación y Demostración (Examen VPN Onion)

¡Hola Pedro! Esta guía te servirá para repasar los conceptos clave para tu presentación de mañana, entender a la perfección qué hizo Denis, y tener un guión de demostración paso a paso estructurado en lenguaje simple.

---

## 🙋‍♂️ 1. ¿Tengo el contexto de nuestra conversación?
**Sí, absolutamente.** Tengo todo el contexto de nuestra conversación desde el principio. Sé que estás trabajando con Denis, que él hizo la captura en C++ y tú hiciste el enrutamiento en Go, que tu máquina es macOS (donde probamos la simulación con Python) y que la máquina de Denis es Linux (donde correrán la demo integrada real).

---

## 🔍 2. ¿Qué significa que "C++ es local y no captura Wi-Fi directamente"?
Denis tiene toda la razón. Su código de C++ **no es un receptor de señales de radio Wi-Fi**. 
* **Cómo funciona en la realidad:**
  * Denis creó una **Tarjeta de Red Virtual** llamada `vpn0`.
  * Cuando tú intentas enviar algo (por ejemplo, haces un `ping` o abres una web), tu computadora le entrega ese paquete a `vpn0`.
  * El C++ de Denis está "escuchando" esa tarjeta virtual localmente dentro de la computadora. Toma los paquetes de ahí y se los da a tu Go.
  * Tu programa de Go (`vpn-core`) es el que **sí envía** esos datos encriptados a través de tu tarjeta de Wi-Fi real hacia el internet.
  * **Analía cotidiana:** C++ es el empleado del correo dentro de tu oficina (local) que recoge las cartas de los escritorios. Go es el camión que viaja por la carretera (Wi-Fi real) para llevar los paquetes a otras oficinas.

---

## 💻 3. Requerimientos para la Demostración de Mañana
¿Necesitan más computadoras? **No. Solo necesitan la computadora de Denis con Linux.**

Aunque el Onion Routing simula servidores en distintas partes del mundo, en programación podemos simular todo en la misma computadora usando **puertos de red locales (localhost)**.
* Correrán el cliente de Go en el puerto `9000`.
* Correrán el Nodo 1 en el puerto `9001`.
* Correrán el Nodo 2 en el puerto `9002`.
* Correrán el Nodo 3 en el puerto `9003`.

**¿Por qué esto es mejor?** El Wi-Fi de las universidades suele ser inestable o bloquea los puertos de comunicación entre laptops. Hacer la demostración completa dentro de la laptop de Denis garantiza un **100% de éxito** sin depender del Wi-Fi de la sala.

---

## 📑 4. Estructura de la Presentación (Guión sugerido)

### Parte A: La Introducción (Tu Speech)
1. **El Problema:** *"En internet, cuando enviamos información, cualquiera en nuestro Wi-Fi puede espiar qué enviamos y a quién (metáfora de enviar una carta sin sobre)".*
2. **Nuestra Solución:** *"Implementamos una VPN de tipo Onion Routing (Enrutamiento de cebolla). Separamos las tareas en dos partes para mayor eficiencia: C++ captura los datos de la máquina y Go encripta y rutea por la red".*
3. **El Concepto Onion (Los sobres):** *"Metemos el paquete en 3 sobres. Cada nodo de la red solo abre un sobre y lo pasa al siguiente. Nadie en el camino conoce a la vez el contenido y el emisor original".*

### Parte B: La Demostración en Vivo (Paso a Paso)

Muestren en el proyector la pantalla de la laptop de Denis con 3 terminales abiertas:

#### **Paso 1: Compilar el proyecto**
En la terminal principal ejecuten:
```bash
make build
```
* **Explicación al profesor:** *"Diseñamos un Makefile global que compila el capturador C++ optimizado y nuestro motor de Go simultáneamente con un solo comando".*

#### **Paso 2: Arrancar la VPN**
* En la **Terminal 1 (Go)** ejecutan:
  ```bash
  ./src/vpn-core/vpn-core -mode all
  ```
* En la **Terminal 2 (C++)** ejecutan:
  ```bash
  sudo ./src/vpn-bridge/vpn-bridge
  ```
* En una **Terminal 3**, configuran la tarjeta de red e inician la prueba de fuego:
  ```bash
  sudo ip addr add 10.8.0.1/24 dev vpn0
  sudo ip link set vpn0 up
  ping -I vpn0 -c 4 10.8.0.5
  ```

#### **Paso 3: Explicar lo que está pasando en pantalla**
Mientras el `ping` responde con éxito, muestren los logs de la **Terminal 1 (Go)** al profesor:
1. *"Aquí ven cómo el cliente en Go recibe el ping de C++ y lo encripta en 3 capas usando AES-GCM con claves ECDH P-256".*
2. *"El paquete viaja cifrado por UDP al puerto 9001 (Nodo 1), luego al 9002 (Nodo 2), y finalmente al 9003 (Nodo 3)".*
3. *"El Nodo 3 (Salida) descifra la última capa y responde al ping. La respuesta vuelve a encriptarse y hace el camino de regreso hacia el cliente".*
4. *"Si alguien estuviera husmeando en la red UDP, solo vería bytes encriptados ilegibles".*
