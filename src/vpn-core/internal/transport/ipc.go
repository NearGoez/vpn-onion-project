package transport

import (
	"fmt"
	"net"
	"os"
)

// IPCServer gestiona la comunicación local a través de un Unix Domain Socket.
type IPCServer struct {
	socketPath string
	conn       *net.UnixConn
	lastClient *net.UnixAddr // Guarda la dirección del último cliente que nos habló
}

// NewIPCServer crea una nueva instancia de nuestro servidor.
func NewIPCServer(socketPath string) *IPCServer {
	return &IPCServer{
		socketPath: socketPath,
	}
}

// Start inicializa el socket UNIX y empieza a escuchar datagramas.
func (s *IPCServer) Start() error {
	// 1. Limpieza preventiva: si el socket quedó de una ejecución fallida anterior, lo borramos.
	if _, err := os.Stat(s.socketPath); err == nil {
		_ = os.Remove(s.socketPath)
	}

	// 2. Escuchar usando el protocolo "unixgram" (datagramas locales SOCK_DGRAM), tal como pide el contrato.
	addr, err := net.ResolveUnixAddr("unixgram", s.socketPath)
	if err != nil {
		return fmt.Errorf("error al resolver la ruta del socket: %w", err)
	}

	conn, err := net.ListenUnixgram("unixgram", addr)
	if err != nil {
		return fmt.Errorf("error al levantar el socket unix: %w", err)
	}

	s.conn = conn
	fmt.Printf("[IPC] Escuchando en el socket local: %s\n", s.socketPath)
	return nil
}

// Listen se queda esperando paquetes continuamente en segundo plano.
// Recibe como argumento una función (handler) que procesará cada paquete recibido.
func (s *IPCServer) Listen(handler func([]byte)) {
	// Creamos un buffer estático de 2048 bytes para recibir paquetes (el MTU real de red es 1500).
	buffer := make([]byte, 2048)

	for {
		// ReadFromUnix se bloquea hasta que llegue un paquete y nos da la dirección del cliente
		n, addr, err := s.conn.ReadFromUnix(buffer)
		if err != nil {
			// Si la conexión se cierra, salimos del bucle.
			break
		}

		if n > 0 {
			// Guardamos la dirección del cliente para poder responderle después
			if addr != nil {
				s.lastClient = addr
			}

			// Copiamos los bytes exactos recibidos para procesarlos
			packet := make([]byte, n)
			copy(packet, buffer[:n])
			handler(packet)
		}
	}
}

// WriteBack envía datos de vuelta al último cliente que nos envió un paquete.
func (s *IPCServer) WriteBack(data []byte) (int, error) {
	if s.lastClient == nil {
		return 0, fmt.Errorf("no hay clientes registrados para enviar datos de bajada")
	}
	return s.conn.WriteToUnix(data, s.lastClient)
}

// Close cierra el descriptor del socket y borra el archivo del sistema para no dejar basura.
func (s *IPCServer) Close() {
	if s.conn != nil {
		s.conn.Close()
	}
	if _, err := os.Stat(s.socketPath); err == nil {
		_ = os.Remove(s.socketPath)
		fmt.Println("[IPC] Archivo de socket eliminado del sistema.")
	}
}
