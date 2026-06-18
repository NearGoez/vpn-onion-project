# 🛠️ Configuración del Compilador y Flags
CXX = g++
CXXFLAGS = -std=c++17 -Wall -Wextra -O3 -I./include

# 📁 Archivos y Directorios
SRC_DIR = src
OBJ_DIR = obj
TARGET = vpn-bridge

# Buscar todos los archivos .cpp en src/
SRCS = $(wildcard $(SRC_DIR)/*.cpp)
# Generar los nombres de los archivos .o equivalentes dentro de obj/
OBJS = $(patsubst $(SRC_DIR)/%.cpp, $(OBJ_DIR)/%.o, $(SRCS))

# 🚀 Reglas Principales
all: $(TARGET)

# Vinculación del binario final
$(TARGET): $(OBJS)
	$(CXX) $(CXXFLAGS) -o $(TARGET) $(OBJS)

# Compilación de objetos individuales (crea el directorio obj si no existe)
$(OBJ_DIR)/%.o: $(SRC_DIR)/%.cpp
	@mkdir -p $(OBJ_DIR)
	$(CXX) $(CXXFLAGS) -c $< -o $@

# 🧹 Limpieza
clean:
	rm -rf $(OBJ_DIR) $(TARGET)

.PHONY: all clean
