#pragma once

#include <string>

class TunDevice {
private:
    int tun_fd;
    std::string device_name;

public:
    TunDevice(const std::string& name = "vpn0");

    ~TunDevice(); // destructor;

    void initialize();

    void close_device(); // cerrar tun

    // obtener el file descriptor
    int get_fd() const { return tun_fd; }
};
