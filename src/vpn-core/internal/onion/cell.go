package onion

import (
	"encoding/binary"
	"fmt"
)

// Definimos los tipos de comando que puede tener una celda en nuestro protocolo Onion.
const (
	CmdCreate  byte = 1 // Crear un nuevo circuito (Handshake inicial)
	CmdCreated byte = 2 // Confirmar creación de circuito (Respuesta al handshake)
	CmdRelay   byte = 3 // Paquete de datos encriptado (IP crudo que viaja por el túnel)
	CmdDestroy byte = 4 // Cerrar el circuito
)

// Cell representa la estructura del sobre (Celda) que viaja por el internet público.
type Cell struct {
	CircuitID uint32 // ID del circuito (para saber a qué túnel pertenece este paquete)
	Command   byte   // Tipo de comando (CmdCreate, CmdRelay, etc.)
	Payload   []byte // Los datos reales (que irán encriptados en capas)
}

// NewCell crea una nueva celda lista para usarse.
func NewCell(circuitID uint32, command byte, payload []byte) *Cell {
	return &Cell{
		CircuitID: circuitID,
		Command:   command,
		Payload:   payload,
	}
}

// Serialize convierte la estructura de la Celda a un arreglo de bytes crudos.
// Formato binario:
// [ 4 bytes: CircuitID ] [ 1 byte: Command ] [ 2 bytes: PayloadLength ] [ Variable: Payload ]
func (c *Cell) Serialize() []byte {
	payloadLen := len(c.Payload)
	// Creamos un buffer con el tamaño exacto del encabezado (7 bytes) más el payload
	buf := make([]byte, 7+payloadLen)

	// 1. Guardamos el CircuitID (4 bytes) en formato BigEndian (estándar de red)
	binary.BigEndian.PutUint32(buf[0:4], c.CircuitID)

	// 2. Guardamos el comando (1 byte)
	buf[4] = c.Command

	// 3. Guardamos la longitud del payload (2 bytes)
	binary.BigEndian.PutUint16(buf[5:7], uint16(payloadLen))

	// 4. Copiamos el payload al final
	copy(buf[7:], c.Payload)

	return buf
}

// Deserialize toma un arreglo de bytes de la red y lo reconstruye en una Celda en memoria.
func Deserialize(data []byte) (*Cell, error) {
	if len(data) < 7 {
		return nil, fmt.Errorf("datos demasiado cortos para ser una celda válida")
	}

	// 1. Leemos el CircuitID
	circuitID := binary.BigEndian.Uint32(data[0:4])

	// 2. Leemos el comando
	command := data[4]

	// 3. Leemos la longitud esperada del payload
	payloadLen := int(binary.BigEndian.Uint16(data[5:7]))

	// Verificación de seguridad para evitar que los datos estén truncados
	if len(data) < 7+payloadLen {
		return nil, fmt.Errorf("la celda está incompleta: esperado %d bytes, obtenido %d", 7+payloadLen, len(data))
	}

	// 4. Extraemos el payload
	payload := make([]byte, payloadLen)
	copy(payload, data[7:7+payloadLen])

	return &Cell{
		CircuitID: circuitID,
		Command:   command,
		Payload:   payload,
	}, nil
}
