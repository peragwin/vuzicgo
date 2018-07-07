import network, socket, time
import gc

class SocketReceiver:
    ifconfig = ('','')

    def __init__(self, ssid='', passwd='', oled=None, skgrid=None):
        self._oled = oled
        self._sta_if = sta_if = network.WLAN(network.STA_IF)
        self._ap_if = ap_if = network.WLAN(network.AP_IF)

        ap_if.active(True)
        sta_if.active(True)

        if not sta_if.isconnected() and ssid:
            sta_if.connect(ssid, passwd)

            timeout = 0
            while not sta_if.isconnected() and timeout < 10:
                print('..', end='')
                time.sleep_ms(500)
                timeout += 1

        if sta_if.isconnected():
            print("Connected to wifi!")
            self.ifconfig = sta_if.ifconfig()
            print(self.ifconfig)
        else:
            sta_if.disconnect()
            sta_if.active(False)
            print("Connection not established..")
            print("Configured in AP mode")
            self.ifconfig = ap_if.ifconfig()
            print(self.ifconfig)

        self._skgrid = skgrid

    def display_ready_to_accept(self):
        oled = self._oled
        oled.fill(0)
        oled.text('ready to accept:', 0, 0)
        oled.text('%s' % self.ifconfig[0], 0, 10)
        oled.text(' on port: 1234', 0, 20)
        oled.show()

    def display_client_connected(self, addr):
        oled = self._oled
        oled.fill(0)
        oled.text('client connected', 0, 0)
        oled.text('%s' % addr[0], 0, 10)
        oled.show()

    def socket_init(self) -> socket.socket:
        # s = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        s = socket.socket()
        s.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
        s.bind(('0.0.0.0', 1234))
        s.listen(1)
        return s

    def listen_skgrid(self):
        s = self.socket_init()

        try:
            while True:
                self.display_ready_to_accept()
                print('ready to accept connection on port 1234')
                # new_connection = True
                cl, addr = s.accept()

                self.display_client_connected(addr)
                print('client connected from', addr)

                count = 0
                last = time.ticks_ms()
                while True:
                    # gc.collect()
                    r = cl.recv(1024)
                    # r, addr = s.recvfrom(1024)
                    # if new_connection:
                    #     self.display_client_connected(addr)
                    #     new_connection = False

                    # check if remote has closed the connection
                    if not r:
                        break
                    self._skgrid.write(r)
                    # try:
                    #     cl.send(chr(0x01))
                    # except:
                    #     break

                    count+=1
                    if count % 200 == 0:
                        ela = time.ticks_ms() - last
                        self._oled.framebuf.fill_rect(5*8, 50, 8*8, 60, 0)
                        self._oled.text("FPS: %d" % (1000 * count / ela), 0, 50)
                        self._oled.show()

                cl.close()

        finally:
            s.close()

    def listen_display(self):
        s = self.socket_init()
        
        try:
            while True:
                self.display_ready_to_accept()

                cl, addr = s.accept()
                print(addr)

                self.display_client_connected(addr)
    
                while True:
                    r = cl.recv(128*8)
                    # check if remote has closed the connection
                    if not r:
                        break

                    oled = self._oled
                    oled.fill(0)
                    # s = str(r, 'utf-8')
                    # for i in range(6):
                    #     t = s[16*i:16*(i+1)]
                    #     if t:
                    #         oled.text(t, 0, 10*i)
                    #     else:
                    #         break
                    for i, b in enumerate(r):
                        for j in range(8):
                            x = (8*i+j) % 128
                            y = (8*i+j) // 128
                            oled.pixel(x, y, (b & (1<<j)) != 0)
                    oled.show()
                cl.close()

        except Exception as e:
            print("socket recv threw:", e)
            s.close()
