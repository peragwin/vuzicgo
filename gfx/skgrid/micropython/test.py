import socket
import time
import math
import sk9822

ADDR = ('192.168.0.172', 1234)

def color(r, g, b, a):
    return (int(r), int(g), int(b), int(a))

def main():
    s = socket.socket()
    s.connect(ADDR)

    # frameAA = bytearray([0xaa] * 1024)
    # frame55 = bytearray([0x55] * 1024)

    grid = sk9822.SKGrid(60, 16, s)

    t = 0
    wt = math.pi * 2 / 180
    ph = 2 * math.pi / 3

    while True:
        time.sleep(.015)
        t += 1

        r = 127 * (1 + math.sin(wt * t))
        g = 127 * (1 + math.sin(wt * t + ph))
        b = 127 * (1 + math.sin(wt * t - ph))
        
        grid.fill(color(r, g, b, 1))
        grid.show()

if __name__ == '__main__':
    main()