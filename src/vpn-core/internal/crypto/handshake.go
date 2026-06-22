package crypto

import (
	"crypto/ecdh"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
)

// GenerateEphemeralKeyPair genera un par de claves (privada y pública) usando la curva P-256.
// La clave pública se devuelve en formato de bytes para que pueda ser enviada fácilmente por la red.
func GenerateEphemeralKeyPair() (*ecdh.PrivateKey, []byte, error) {
	// Generamos una clave privada efímera (de un solo uso) usando la curva elíptica P-256
	privKey, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("error al generar clave privada ecdh: %w", err)
	}

	// Extraemos la clave pública asociada y la convertimos a un arreglo de bytes
	pubKeyBytes := privKey.PublicKey().Bytes()

	return privKey, pubKeyBytes, nil
}

// DeriveSharedSecret calcula el secreto compartido usando tu clave privada y la clave pública del otro nodo.
// Luego, usa SHA-256 para derivar una clave simétrica de 32 bytes (256 bits) para encriptar con AES.
func DeriveSharedSecret(myPrivKey *ecdh.PrivateKey, peerPubKeyBytes []byte) ([]byte, error) {
	// 1. Reconstruimos la clave pública del compañero a partir de los bytes recibidos
	peerPubKey, err := ecdh.P256().NewPublicKey(peerPubKeyBytes)
	if err != nil {
		return nil, fmt.Errorf("clave pública del compañero inválida: %w", err)
	}

	// 2. Realizamos la multiplicación matemática de la curva elíptica (ECDH)
	// Esto genera un secreto compartido único que ambos extremos calcularán idénticamente
	sharedSecret, err := myPrivKey.ECDH(peerPubKey)
	if err != nil {
		return nil, fmt.Errorf("error al calcular secreto compartido ecdh: %w", err)
	}

	// 3. Derivamos la clave simétrica final aplicando SHA-256 al secreto compartido.
	// Esto nos asegura obtener exactamente 32 bytes de alta entropía, listos para AES-256.
	derivedKey := sha256.Sum256(sharedSecret)

	return derivedKey[:], nil
}
