package onion

import (
	"bytes"
	"crypto/rand"
	"testing"
)

func TestOnionRoutingSimulation(t *testing.T) {
	// 1. Simulación: Generamos 3 claves simétricas aleatorias de 32 bytes para nuestro circuito.
	// Cada llave representa la que acordamos con un Nodo diferente (1, 2 y 3).
	key1 := make([]byte, 32)
	key2 := make([]byte, 32)
	key3 := make([]byte, 32)
	
	_, _ = rand.Read(key1)
	_, _ = rand.Read(key2)
	_, _ = rand.Read(key3)

	keys := [][]byte{key1, key2, key3}

	// Creamos el circuito en el cliente
	circuit := NewCircuitState(100, keys)

	// Paquete original que simula ser interceptado desde la interfaz TUN de la computadora
	originalPacket := []byte("PAQUETE_IP_CRUDO_DE_PRUEBA")
	t.Logf("1. Paquete original: '%s' (%d bytes)", string(originalPacket), len(originalPacket))

	// ============================================
	// CAMINO DE SUBIDA (CLIENTE -> INTERNET)
	// ============================================
	t.Log("\n--- CAMINO DE SUBIDA ---")

	// El cliente cifra el paquete original en 3 capas
	onionEncrypted, err := circuit.EncryptOnion(originalPacket)
	if err != nil {
		t.Fatalf("Error en el cifrado onion: %v", err)
	}
	t.Logf("2. Paquete cifrado en 3 capas por el cliente: (HEX): %x... (Total: %d bytes)", onionEncrypted[:15], len(onionEncrypted))

	// Nodo 1 recibe el paquete y le pela la primera capa (Capa exterior)
	layer1Peeled, err := PeelLayer(onionEncrypted, key1)
	if err != nil {
		t.Fatalf("Nodo 1 falló al pelar su capa: %v", err)
	}
	t.Logf("3. Nodo 1 pela capa 1 (HEX): %x...", layer1Peeled[:15])

	// Nodo 2 recibe la salida del Nodo 1 y pela la segunda capa
	layer2Peeled, err := PeelLayer(layer1Peeled, key2)
	if err != nil {
		t.Fatalf("Nodo 2 falló al pelar su capa: %v", err)
	}
	t.Logf("4. Nodo 2 pela capa 2 (HEX): %x...", layer2Peeled[:15])

	// Nodo 3 (Nodo de salida) recibe la salida del Nodo 2 y pela la última capa
	layer3Peeled, err := PeelLayer(layer2Peeled, key3)
	if err != nil {
		t.Fatalf("Nodo 3 falló al pelar su capa: %v", err)
	}
	t.Logf("5. Nodo 3 (Salida) pela capa 3. Mensaje recuperado: '%s'", string(layer3Peeled))

	// Verificación de subida
	if !bytes.Equal(originalPacket, layer3Peeled) {
		t.Errorf("Error: El paquete final de subida no coincide con el original")
	}

	// ============================================
	// CAMINO DE BAJADA (INTERNET -> CLIENTE)
	// ============================================
	t.Log("\n--- CAMINO DE BAJADA ---")

	// Supongamos que la respuesta del servidor es el mismo paquete de vuelta.
	// Cada nodo de la red, en reversa, le añade su capa de cifrado:
	
	// Nodo 3 cifra con la Llave 3
	downstreamLayer3, err := WrapLayer(originalPacket, key3)
	if err != nil {
		t.Fatalf("Nodo 3 falló al cifrar bajada: %v", err)
	}

	// Nodo 2 le añade su capa con la Llave 2
	downstreamLayer2, err := WrapLayer(downstreamLayer3, key2)
	if err != nil {
		t.Fatalf("Nodo 2 falló al cifrar bajada: %v", err)
	}

	// Nodo 1 le añade su capa con la Llave 1
	downstreamLayer1, err := WrapLayer(downstreamLayer2, key1)
	if err != nil {
		t.Fatalf("Nodo 1 falló al cifrar bajada: %v", err)
	}

	t.Logf("6. Paquete de respuesta cifrado en 3 capas de regreso al cliente: (HEX): %x...", downstreamLayer1[:15])

	// El cliente recibe el paquete encriptado y le quita las 3 capas a la vez
	decryptedResponse, err := circuit.DecryptOnion(downstreamLayer1)
	if err != nil {
		t.Fatalf("Cliente falló al descifrar respuesta onion: %v", err)
	}
	t.Logf("7. Cliente descifró la respuesta onion: '%s'", string(decryptedResponse))

	// Verificación de bajada
	if !bytes.Equal(originalPacket, decryptedResponse) {
		t.Errorf("Error: El paquete final descifrado por el cliente no coincide con el original")
	}
}
