package main

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"vpn-core/internal/crypto"
	"vpn-core/internal/onion"
	"vpn-core/internal/transport"
)

// Claves simétricas estáticas precompartidas para la demostración
var (
	key1 = sha256.Sum256([]byte("nodo1"))
	key2 = sha256.Sum256([]byte("nodo2"))
	key3 = sha256.Sum256([]byte("nodo3"))
	keys = [][]byte{key1[:], key2[:], key3[:]}
)

func main() {
	// Definimos banderas para poder ejecutar el binario en diferentes modos
	mode := flag.String("mode", "all", "Modo de ejecución: all (simulador), client (cliente VPN), node1, node2, node3")
	flag.Parse()

	fmt.Printf("[SYSTEM] Iniciando en Modo: %s\n", *mode)

	// Canal para escuchar señales de apagado (Ctrl+C)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	switch *mode {
	case "client":
		runClient(sigChan)
	case "node1":
		runNode(1, 9001, "127.0.0.1:9002", "127.0.0.1:9000", key1[:], sigChan)
	case "node2":
		runNode(2, 9002, "127.0.0.1:9003", "127.0.0.1:9001", key2[:], sigChan)
	case "node3":
		runExitNode(9003, "127.0.0.1:9002", key3[:], sigChan)
	case "all":
		// Modo simulador: Corre todo en segundo plano en un solo proceso
		fmt.Println("[ALL-IN-ONE] Levantando toda la red Onion localmente...")
		
		go runExitNode(9003, "127.0.0.1:9002", key3[:], nil)
		go runNode(2, 9002, "127.0.0.1:9003", "127.0.0.1:9001", key2[:], nil)
		go runNode(1, 9001, "127.0.0.1:9002", "127.0.0.1:9000", key1[:], nil)
		
		// Levantamos el servidor web interactivo para la presentación
		go startWebServer()

		// El hilo principal ejecuta el cliente
		runClient(sigChan)
	default:
		fmt.Println("[ERROR] Modo desconocido. Usa: -mode [all|client|node1|node2|node3]")
	}
}

// runClient inicializa el cliente VPN.
// Lee paquetes del socket local C++, los encripta en 3 capas, los envía a Node 1.
// También recibe respuestas de Node 1, las descifra y las devuelve al socket C++.
func runClient(sigChan chan os.Signal) {
	socketPath := "/tmp/onion_vpn.sock"
	ipcServer := transport.NewIPCServer(socketPath)

	err := ipcServer.Start()
	if err != nil {
		fmt.Printf("[CLIENT CRITICAL] Error socket local: %v\n", err)
		os.Exit(1)
	}

	// UDP para recibir respuestas de la red Onion (Puerto 9000)
	udpTransport, err := transport.NewUDPTransport(9000)
	if err != nil {
		fmt.Printf("[CLIENT CRITICAL] Error UDP: %v\n", err)
		ipcServer.Close()
		os.Exit(1)
	}

	circuit := onion.NewCircuitState(100, keys)

	// Goroutine 1: Recibe del Socket C++ (Subida) -> Cifra en 3 capas -> Envía a Node 1 (9001)
	go ipcServer.Listen(func(packet []byte) {
		if len(packet) == 0 {
			return
		}

		// Si es un paquete de texto para el demo
		isText := packet[0] >= 32 && packet[0] <= 126
		if isText {
			fmt.Printf("[CLIENT -> UP] Recibido mensaje del socket: '%s'\n", string(packet))
		} else {
			fmt.Printf("[CLIENT -> UP] Recibido paquete IP de %d bytes del socket.\n", len(packet))
		}

		// Encriptamos el paquete en 3 capas (Onion)
		encryptedPayload, err := circuit.EncryptOnion(packet)
		if err != nil {
			fmt.Printf("[CLIENT ERROR] Error al cifrar: %v\n", err)
			return
		}

		// Creamos la Celda
		cell := onion.NewCell(circuit.CircuitID, onion.CmdRelay, encryptedPayload)
		cellBytes := cell.Serialize()

		// Enviamos al Nodo 1 (Puerto 9001)
		err = udpTransport.Send("127.0.0.1:9001", cellBytes)
		if err != nil {
			fmt.Printf("[CLIENT ERROR] Error al enviar a Nodo 1: %v\n", err)
		} else {
			fmt.Printf("[CLIENT -> UP] Celda cifrada enviada a Nodo 1 (Puerto 9001). Tamaño celda: %d bytes\n", len(cellBytes))
		}
	})

	// Goroutine 2: Recibe de Node 1 (Bajada) -> Descifra 3 capas -> Envía al Socket C++
	go udpTransport.Listen(func(data []byte, addr *net.UDPAddr) {
		cell, err := onion.Deserialize(data)
		if err != nil {
			fmt.Printf("[CLIENT ERROR] Celda inválida recibida: %v\n", err)
			return
		}

		if cell.Command == onion.CmdRelay {
			// Desciframos las 3 capas
			decryptedPayload, err := circuit.DecryptOnion(cell.Payload)
			if err != nil {
				fmt.Printf("[CLIENT ERROR] Error al descifrar cebolla: %v\n", err)
				return
			}

			// Enviamos los bytes limpios de vuelta a C++
			_, err = ipcServer.WriteBack(decryptedPayload)
			if err != nil {
				fmt.Printf("[CLIENT ERROR] Error al reinyectar al socket C++: %v\n", err)
			} else {
				isText := decryptedPayload[0] >= 32 && decryptedPayload[0] <= 126
				if isText {
					fmt.Printf("[CLIENT -> DOWN] Mensaje descifrado reinyectado a C++: '%s'\n", string(decryptedPayload))
				} else {
					fmt.Printf("[CLIENT -> DOWN] Paquete IP descifrado de %d bytes reinyectado a C++.\n", len(decryptedPayload))
				}
			}
		}
	})

	fmt.Println("[CLIENT] Cliente VPN listo. Presiona Ctrl+C para salir...")
	
	if sigChan != nil {
		sig := <-sigChan
		fmt.Printf("\n[CLIENT] Señal recibida: %v. Apagando...\n", sig)
		ipcServer.Close()
		udpTransport.Close()
	}
}

// runNode inicializa los nodos intermedios (Nodo 1 y Nodo 2)
func runNode(nodeNum int, port int, nextHop string, prevHop string, key []byte, sigChan chan os.Signal) {
	udpTransport, err := transport.NewUDPTransport(port)
	if err != nil {
		fmt.Printf("[NODE %d CRITICAL] Error: %v\n", nodeNum, err)
		return
	}

	go udpTransport.Listen(func(data []byte, addr *net.UDPAddr) {
		cell, err := onion.Deserialize(data)
		if err != nil {
			return
		}

		// Determinamos la dirección del tráfico basándonos en quién nos envió el paquete
		senderAddr := addr.String()
		isUpstream := senderAddr != nextHop // Si no viene del siguiente nodo, viene del anterior (subida)

		if isUpstream {
			// Subida: Pelamos una capa y enviamos al siguiente
			peeledPayload, err := onion.PeelLayer(cell.Payload, key)
			if err != nil {
				fmt.Printf("[NODE %d ERROR] Fallo al pelar capa: %v\n", nodeNum, err)
				return
			}
			fmt.Printf("[NODE %d -> UP] Pelada capa exterior. Reenviando al siguiente salto: %s\n", nodeNum, nextHop)
			
			cell.Payload = peeledPayload
			udpTransport.Send(nextHop, cell.Serialize())
		} else {
			// Bajada: Agregamos nuestra capa y enviamos al anterior
			wrappedPayload, err := onion.WrapLayer(cell.Payload, key)
			if err != nil {
				fmt.Printf("[NODE %d ERROR] Fallo al envolver capa: %v\n", nodeNum, err)
				return
			}
			fmt.Printf("[NODE %d -> DOWN] Agregada capa de cifrado. Reenviando al salto anterior: %s\n", nodeNum, prevHop)
			
			cell.Payload = wrappedPayload
			udpTransport.Send(prevHop, cell.Serialize())
		}
	})

	if sigChan != nil {
		<-sigChan
		udpTransport.Close()
	}
}

// runExitNode inicializa el Nodo 3 (Exit Node).
// Recibe el paquete, le quita la última capa, procesa los datos (o hace un loopback simulado)
// y envía el paquete de vuelta hacia abajo por el circuito.
func runExitNode(port int, prevHop string, key []byte, sigChan chan os.Signal) {
	udpTransport, err := transport.NewUDPTransport(port)
	if err != nil {
		fmt.Printf("[EXIT NODE CRITICAL] Error: %v\n", err)
		return
	}

	go udpTransport.Listen(func(data []byte, addr *net.UDPAddr) {
		cell, err := onion.Deserialize(data)
		if err != nil {
			return
		}

		// El Nodo de salida solo recibe paquetes de subida desde el Nodo 2
		peeledPayload, err := onion.PeelLayer(cell.Payload, key)
		if err != nil {
			fmt.Printf("[EXIT NODE ERROR] Fallo al pelar capa: %v\n", err)
			return
		}

		// 1. Verificamos de forma estricta si es un paquete IP real.
		// Para IPv4: el primer byte empieza con 4 y el tamaño real del buffer coincide con el campo 'Total Length' de la cabecera IP (bytes 2 y 3).
		isIPv4 := len(peeledPayload) >= 20 && (peeledPayload[0]>>4) == 4 && int(binary.BigEndian.Uint16(peeledPayload[2:4])) == len(peeledPayload)
		// Para IPv6: el primer byte empieza con 6 y el tamaño real coincide con la longitud de carga útil + 40 bytes de cabecera.
		isIPv6 := len(peeledPayload) >= 40 && (peeledPayload[0]>>4) == 6 && int(binary.BigEndian.Uint16(peeledPayload[4:6]))+40 == len(peeledPayload)
		isIP := isIPv4 || isIPv6
		
		var responseBytes []byte

		if isIP {
			fmt.Printf("[EXIT NODE] ¡PAQUETE IP DESCIFRADO COMPLETAMENTE! Tamaño: %d bytes. Aplicando Loopback IP...\n", len(peeledPayload))
			
			// Si es un paquete IP (ej. un ping), hacemos el loopback real convirtiendo el Request en Reply
			responseBytes = handleIPLoopback(peeledPayload)
		} else {
			fmt.Printf("[EXIT NODE] ¡MENSAJE DESCIFRADO COMPLETAMENTE! Contenido: '%s'\n", string(peeledPayload))
			
			// Para mensajes de texto de prueba, devolvemos el texto original intacto y limpio
			responseBytes = peeledPayload
		}

		// Encriptamos la respuesta con nuestra llave (Llave 3) para iniciar el camino de regreso
		wrappedPayload, err := onion.WrapLayer(responseBytes, key)
		if err != nil {
			return
		}

		cell.Payload = wrappedPayload
		
		// Enviamos de vuelta al Nodo 2
		fmt.Printf("[EXIT NODE -> DOWN] Respuesta cifrada. Reenviando al nodo anterior: %s\n", prevHop)
		udpTransport.Send(prevHop, cell.Serialize())
	})

	if sigChan != nil {
		<-sigChan
		udpTransport.Close()
	}
}

// handleIPLoopback modifica un paquete IP ICMP Echo Request para convertirlo en ICMP Echo Reply.
// Esto permite que el comando 'ping' crea que el destino le respondió de verdad.
func handleIPLoopback(packet []byte) []byte {
	if len(packet) < 20 {
		return packet
	}

	// Intercambiamos IPs: Origen (bytes 12-15) por Destino (bytes 16-19)
	srcIP := make([]byte, 4)
	copy(srcIP, packet[12:16])
	copy(packet[12:16], packet[16:20])
	copy(packet[16:20], srcIP)

	// Si el protocolo de capa 3 es ICMP (1)
	if packet[9] == 1 && len(packet) >= 28 {
		// Convertimos ICMP Echo Request (Tipo 8) a ICMP Echo Reply (Tipo 0)
		if packet[20] == 8 {
			packet[20] = 0 // Tipo 0

			// Recalculamos el checksum de ICMP para que la respuesta sea válida en la red
			packet[22] = 0
			packet[23] = 0
			csum := ipChecksum(packet[20:])
			binary.BigEndian.PutUint16(packet[22:24], csum)
		}
	}

	return packet
}

// ipChecksum calcula el checksum estándar de red IP/ICMP
func ipChecksum(data []byte) uint16 {
	var sum uint32
	for i := 0; i < len(data)-1; i += 2 {
		sum += uint32(binary.BigEndian.Uint16(data[i : i+2]))
	}
	if len(data)%2 == 1 {
		sum += uint32(data[len(data)-1]) << 8
	}
	for sum > 0xffff {
		sum = (sum & 0xffff) + (sum >> 16)
	}
	return uint16(^sum)
}

// startWebServer levanta un servidor HTTP local que sirve la interfaz web de la demo.
func startWebServer() {
	// Servimos la página web estática
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "cmd/index.html")
	})

	// API que simula el cifrado Onion multicapa en tiempo real y retorna los logs del circuito
	http.HandleFunc("/api/simulate", func(w http.ResponseWriter, r *http.Request) {
		message := r.URL.Query().Get("message")
		if message == "" {
			message = "Mensaje vacío"
		}

		// 1. Cifrado en el cliente (de atrás hacia adelante: 3 -> 2 -> 1)
		c3, _ := crypto.Encrypt(key3[:], []byte(message))
		c2, _ := crypto.Encrypt(key2[:], c3)
		c1, _ := crypto.Encrypt(key1[:], c2)

		// 2. Descifrado en los nodos (del primer nodo al último)
		d1, _ := onion.PeelLayer(c1, key1[:])
		d2, _ := onion.PeelLayer(d1, key2[:])
		d3, _ := onion.PeelLayer(d2, key3[:])

		steps := []string{
			fmt.Sprintf("[CLIENTE] Capturado mensaje local: '%s'", message),
			fmt.Sprintf("[CLIENTE] Cifrado con Llave 3 (Exit) -> HEX: %x...", c3[:12]),
			fmt.Sprintf("[CLIENTE] Cifrado con Llave 2 (Medio) -> HEX: %x...", c2[:12]),
			fmt.Sprintf("[CLIENTE] Cifrado con Llave 1 (Entrada) -> HEX: %x...", c1[:12]),
			fmt.Sprintf("[NODO 1] Celda recibida. Descifrada Capa 1. Reenviando (HEX: %x...)", d1[:12]),
			fmt.Sprintf("[NODO 2] Celda recibida. Descifrada Capa 2. Reenviando (HEX: %x...)", d2[:12]),
			fmt.Sprintf("[EXIT NODE] Celda recibida. Descifrada capa 3 final."),
			fmt.Sprintf("[EXIT NODE] ¡Mensaje de origen descifrado con éxito!: '%s'", string(d3)),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string][]string{
			"safe_steps": steps,
		})
	})

	fmt.Println("[WEB] Dashboard interactivo listo en http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Printf("[WEB ERROR] No se pudo levantar el servidor web: %v\n", err)
	}
}

