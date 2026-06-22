package onion

import (
	"fmt"
	"vpn-core/internal/crypto"
)

// CircuitState representa el estado del circuito Onion del cliente.
// Guarda las claves de cifrado acordadas con cada uno de los nodos del circuito.
type CircuitState struct {
	CircuitID uint32
	Keys      [][]byte // Arreglo de llaves: Keys[0] es del Nodo 1, Keys[1] del Nodo 2, Keys[2] del Nodo 3
}

// NewCircuitState crea un nuevo estado de circuito con sus llaves de sesión.
func NewCircuitState(circuitID uint32, keys [][]byte) *CircuitState {
	return &CircuitState{
		CircuitID: circuitID,
		Keys:      keys,
	}
}

// EncryptOnion aplica las capas de cifrado sobre el paquete original (Subida).
// En Onion Routing, el cliente cifra de atrás hacia adelante:
// Primero con la llave del Nodo 3 (el más lejano), luego con la del Nodo 2, y al final con la del Nodo 1.
func (c *CircuitState) EncryptOnion(plaintext []byte) ([]byte, error) {
	currentPayload := plaintext
	var err error

	// Recorremos las llaves en orden inverso (de la última a la primera)
	for i := len(c.Keys) - 1; i >= 0; i-- {
		currentPayload, err = crypto.Encrypt(c.Keys[i], currentPayload)
		if err != nil {
			return nil, fmt.Errorf("error al cifrar capa %d de la cebolla: %w", i+1, err)
		}
	}

	return currentPayload, nil
}

// DecryptOnion remueve secuencialmente las capas de cifrado aplicadas por la red (Bajada).
// Cuando el paquete regresa de internet, viene cifrado por cada nodo en el camino.
// El cliente debe pelar las capas en orden: primero con la Llave 1, luego con la Llave 2 y al final con la Llave 3.
func (c *CircuitState) DecryptOnion(ciphertext []byte) ([]byte, error) {
	currentPayload := ciphertext
	var err error

	// Recorremos las llaves en orden directo (del primer nodo al último)
	for i := 0; i < len(c.Keys); i++ {
		currentPayload, err = crypto.Decrypt(c.Keys[i], currentPayload)
		if err != nil {
			return nil, fmt.Errorf("error al descifrar capa %d de la cebolla: %w", i+1, err)
		}
	}

	return currentPayload, nil
}

// --- FUNCIONES PARA LOS NODOS INTERMEDIOS (PROXIES) ---

// PeelLayer es usada por un nodo intermedio para remover SU capa de cifrado (Subida).
// Al hacerlo, revela la celda encriptada para el siguiente nodo.
func PeelLayer(payload []byte, key []byte) ([]byte, error) {
	return crypto.Decrypt(key, payload)
}

// WrapLayer es usada por un nodo intermedio para agregar SU capa de cifrado (Bajada)
// antes de pasar el paquete hacia el nodo anterior de regreso al cliente.
func WrapLayer(payload []byte, key []byte) ([]byte, error) {
	return crypto.Encrypt(key, payload)
}
