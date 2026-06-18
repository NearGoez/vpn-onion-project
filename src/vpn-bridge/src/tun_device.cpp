
#include "tun_device.hpp"
#include "../include/tun_device.hpp"

#include <iostream>
#include <stdexcept>
#include <cstring>
#include <fcntl.h>
#include <unistd.h>
#include <sys/ioctl.h>
#include <linux/if_tun.h>
#include <net/if.h>


TunDevice::TunDevice(const std::string& name) : tun_fd(-1), device_name(name) {}


TunDevice::~TunDevice() {
    close_device();
}

void TunDevice::initialize() {
    
    tun_fd = open("/dev/net/tun", O_RDWR);
    if (tun_fd < 0) {
        throw std::runtime_error("Error abriendo /dev/net/tun. Tienes privilegios?");
    }

    struct ifreq ifr;
    
    // dejamos todo en cero por si acaso
    std::memset(&ifr, 0, sizeof(ifr));

    // configuramos flags segun eel contrato IPC

    ifr.ifr_flags = IFF_TUN | IFF_NO_PI;

    if (!device_name.empty()) {
        std::strncpy(ifr.ifr_name, device_name.c_str(), IFNAMSIZ);
    }

    int err = ioctl(tun_fd, TUNSETIFF, (void*)&ifr);

    if (err < 0) {
        close(tun_fd);
        tun_fd = -1;
        throw std::runtime_error("Error: Llamada ioctl(TUNSETIFF) fallida al configurar " + device_name);
    }

    device_name = ifr.ifr_name;
    std::cout << "[TUN] Intrfaz '" << device_name << "' creada exitosamente.";
    std::cout << "FD: " << tun_fd << std::endl;
}

void TunDevice::close_device() {
    if (tun_fd >= 0) {
        close(tun_fd);
        std::cout << "[TUN] Interfaz: " << device_name;
        std::cout << "destruida y descriptor cerrado" << std::endl;
        tun_fd = -1;
    }
}











