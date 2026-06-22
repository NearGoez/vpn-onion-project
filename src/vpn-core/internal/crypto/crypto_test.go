package crypto

import (
	"bytes"
	"testing"
)

func TestDiffieHellmanAndEncryption(t *testing.T) {
	// 1. Simulación de Alice (Cliente VPN) y Bob (Nodo de la Red VPN)
	// Cada uno genera su par de llaves públicas y privadas efímeras.
	alicePriv, alicePubBytes, err := GenerateEphemeralKeyPair()
	if err != nil {
		t.Fatalf("Alice falló al generar llaves: %v", err)
	}

	bobPriv, bobPubBytes, err := GenerateEphemeralKeyPair()
	if err != nil {
		t.Fatalf("Bob falló al generar llaves: %v", err)
	}

	// 2. Intercambio de llaves públicas:
	// Alice recibe la llave de Bob, y Bob recibe la de Alice.
	// Cada uno calcula por separado el secreto compartido.
	aliceKey, err := DeriveSharedSecret(alicePriv, bobPubBytes)
	if err != nil {
		t.Fatalf("Alice falló al derivar el secreto compartido: %v", err)
	}

	bobKey, err := DeriveSharedSecret(bobPriv, alicePubBytes)
	if err != nil {
		t.Fatalf("Bob falló al derivar el secreto compartido: %v", err)
	}

	// 3. Verificación de coincidencia matemática:
	// Las llaves derivadas simétricas de 32 bytes deben ser EXACTAMENTE iguales.
	if !bytes.Equal(aliceKey, bobKey) {
		t.Errorf("¡Las llaves simétricas derivadas no coinciden!")
	} else {
		t.Logf("Éxito: Ambas llaves coinciden matemáticamente. Llave derivada (HEX): %x", aliceKey)
	}

	// 4. Prueba de Cifrado y Descifrado:
	// Alice encripta un paquete (por ejemplo, un ping de red simulado)
	originalMessage := []byte("Paquete de datos ultra secreto de Pedro")
	encryptedMsg, err := Encrypt(aliceKey, originalMessage)
	if err != nil {
		t.Fatalf("Fallo al encriptar: %v", err)
	}

	t.Logf("Mensaje encriptado (HEX con Nonce): %x", encryptedMsg)

	// Bob descifra el mensaje usando la llave que él derivó
	decryptedMsg, err := Decrypt(bobKey, encryptedMsg)
	if err != nil {
		t.Fatalf("Fallo al desencriptar: %v", err)
	}

	// Verificamos que el mensaje recuperado sea el original
	if !bytes.Equal(originalMessage, decryptedMsg) {
		t.Errorf("El mensaje desencriptado no coincide con el original. Obtenido: %s", string(decryptedMsg))
	} else {
		t.Logf("Éxito: Mensaje descifrado correctamente: %s", string(decryptedMsg))
	}

	// 5. Prueba de Seguridad (Detección de Manipulación / Tampering):
	// Simulamos que un atacante en la red intercepta el paquete cifrado y altera el último bit.
	t.Log("Simulando alteración de datos por un atacante en la red...")
	corruptedMsg := make([]byte, len(encryptedMsg))
	copy(corruptedMsg, encryptedMsg)
	corruptedMsg[len(corruptedMsg)-1] ^= 0x01 // Volteamos un bit al final

	// Intentamos desencriptar el mensaje alterado
	_, err = Decrypt(bobKey, corruptedMsg)
	if err == nil {
		t.Error("¡PELIGRO! El descifrado del paquete corrupto pasó sin detectar la alteración.")
	} else {
		t.Logf("Éxito de Seguridad: La alteración fue detectada y el descifrado fue bloqueado. Error: %v", err)
	}
}
