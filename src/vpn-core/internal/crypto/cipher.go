package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
)

const (
	// GCMNonceSize es el tamaño estándar recomendado para el vector de inicialización (nonce) de AES-GCM.
	GCMNonceSize = 12
)

// Encrypt encripta un mensaje (plaintext) usando una clave simétrica de 32 bytes con AES-GCM.
// Retorna un arreglo de bytes que contiene el Nonce de 12 bytes pegado al inicio del mensaje cifrado.
func Encrypt(key []byte, plaintext []byte) ([]byte, error) {
	// 1. Instanciamos el cifrador por bloques AES
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("error al crear cifrador aes: %w", err)
	}

	// 2. Envolvemos el cifrador en modo GCM (Galois/Counter Mode) para cifrado autenticado (AEAD)
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("error al iniciar modo gcm: %w", err)
	}

	// 3. Generamos un Nonce único y aleatorio de 12 bytes.
	// ADVERTENCIA DE SEGURIDAD: Nunca debemos reutilizar un mismo Nonce con la misma clave.
	nonce := make([]byte, GCMNonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("error al generar nonce aleatorio: %w", err)
	}

	// 4. Ciframos el mensaje. 
	// aesGCM.Seal(dest, nonce, plaintext, dataAsociada)
	// Al pasarle 'nonce' como primer argumento, Go escribe automáticamente el Nonce al inicio del resultado
	// y luego le concatena los bytes cifrados y la firma de autenticidad.
	ciphertext := aesGCM.Seal(nonce, nonce, plaintext, nil)

	return ciphertext, nil
}

// Decrypt recibe el paquete cifrado (con el Nonce al inicio) y lo descifra usando la clave simétrica.
// Si el paquete fue alterado en tránsito, la autenticación fallará y devolverá un error.
func Decrypt(key []byte, ciphertextWithNonce []byte) ([]byte, error) {
	if len(ciphertextWithNonce) < GCMNonceSize {
		return nil, fmt.Errorf("datos cifrados demasiado cortos para contener el nonce")
	}

	// 1. Instanciamos el cifrador AES
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("error al crear cifrador aes: %w", err)
	}

	// 2. Iniciamos el modo GCM
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("error al iniciar modo gcm: %w", err)
	}

	// 3. Separamos el Nonce de 12 bytes del inicio y el resto de los bytes cifrados
	nonce := ciphertextWithNonce[:GCMNonceSize]
	ciphertext := ciphertextWithNonce[GCMNonceSize:]

	// 4. Desciframos y verificamos la firma de autenticidad
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("error al descifrar (¿datos alterados o clave incorrecta?): %w", err)
	}

	return plaintext, nil
}
