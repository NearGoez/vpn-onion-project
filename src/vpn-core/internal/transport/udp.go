package transport

import (
	"fmt"
	"net"
)

// UDPTransport gestiona el envío y recepción de paquetes cifrados a través del internet público.
type UDPTransport struct {
	conn *net.UDPConn
}

// NewUDPTransport inicializa un socket de escucha UDP en el puerto asignado.
func NewUDPTransport(port int) (*UDPTransport, error) {
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("0.0.0.0:%d", port))
	if err != nil {
		return nil, fmt.Errorf("error al resolver dirección udp: %w", err)
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, fmt.Errorf("error al escuchar en puerto udp: %w", err)
	}

	fmt.Printf("[UDP] Escuchando tráfico público en puerto :%d\n", port)
	return &UDPTransport{conn: conn}, nil
}

// Send envía un arreglo de bytes (celda cifrada) a un destino UDP (la IP y puerto del siguiente salto).
func (u *UDPTransport) Send(targetAddr string, data []byte) error {
	addr, err := net.ResolveUDPAddr("udp", targetAddr)
	if err != nil {
		return fmt.Errorf("error al resolver dirección destino: %w", err)
	}

	_, err = u.conn.WriteToUDP(data, addr)
	if err != nil {
		return fmt.Errorf("error al enviar por udp: %w", err)
	}

	return nil
}

// Listen corre un bucle que se queda leyendo datagramas del socket UDP.
// Envía los bytes recibidos y la dirección del remitente al handler correspondiente.
func (u *UDPTransport) Listen(handler func([]byte, *net.UDPAddr)) {
	buffer := make([]byte, 2048)

	for {
		n, addr, err := u.conn.ReadFromUDP(buffer)
		if err != nil {
			// Si la conexión se cierra, salimos del bucle.
			break
		}

		if n > 0 {
			packet := make([]byte, n)
			copy(packet, buffer[:n])
			handler(packet, addr)
		}
	}
}

// Close cierra la conexión del socket UDP.
func (u *UDPTransport) Close() {
	if u.conn != nil {
		u.conn.Close()
		fmt.Println("[UDP] Socket UDP cerrado.")
	}
}
