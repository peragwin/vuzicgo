import json
from machine import I2C, SPI, Pin
import ssd1306
from sk9822 import SK9822
import srecv

try:
    with open('.wifi', mode='rb') as f:
        wifi = json.load(f)
except OSError:
    wifi = {}

i2c = I2C(scl=Pin(15), sda=Pin(4))
oled = ssd1306.SSD1306_I2C(128, 64, i2c, Pin(16, Pin.OUT))
spi = SPI(1, baudrate=4000000, polarity=0, phase=1,
            sck=Pin(14), mosi=Pin(13))
sk = SK9822(16*60, spi)

_s = "SK9822 Grid WIFI controller initializing..."
for i in range(len(_s)/16):
    oled.text(_s[16*i:16*(i+1)], 0, 10*i)
oled.show()
del _s

rcv = srecv.SocketReceiver(
    ssid=wifi.get('ssid', ''), passwd=wifi.get('passwd', ''),
    oled=oled, skgrid=sk)
rcv.listen_skgrid()