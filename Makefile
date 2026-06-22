# 🚀 Orquestador de Compilación Global del Proyecto

.PHONY: all build clean test-go

# Regla por defecto al ejecutar 'make': compila todo
all: build

# Compila ambos componentes del monorepo
build:
	@echo "🛠️ Compilando vpn-bridge (C++)..."
	$(MAKE) -C src/vpn-bridge
	@echo "🛠️ Compilando vpn-core (Go)..."
	cd src/vpn-core && go build -o vpn-core cmd/main.go
	@echo "🎉 ¡Compilación global completada con éxito!"

# Limpia los binarios compilados y archivos basura de ambos proyectos
clean:
	@echo "🧹 Limpiando archivos compilados de C++..."
	$(MAKE) -C src/vpn-bridge clean
	@echo "🧹 Limpiando archivos compilados de Go..."
	rm -f src/vpn-core/vpn-core
	@echo "✨ Workspace limpio."

# Corre todas las pruebas unitarias de Go
test-go:
	@echo "🧪 Ejecutando suite de pruebas unitarias de Go..."
	cd src/vpn-core && go test -v ./...
