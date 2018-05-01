
class SK9822:
    def __init__(self, count: int, device):
        self.count = count
        self.framesize = 4*count
        self.startframe = bytearray(4)
        self.endframe = bytearray(6 + int(count/16))
        self.endframe[0] = 0xff
        self._device = device
        self.buffersize = 4 + self.framesize + len(self.endframe)

    def show(self, buffer: bytearray):
        self._device.write(self.startframe + buffer + self.endframe)

    def write(self, buffer: bytearray):
        self._device.write(buffer)

class SKGrid(SK9822):
    def __init__(self, width: int, height: int, device):
        self.width = width
        self.height = height
        count = width * height
        super().__init__(count, device)
        self._buffer = bytearray(4 + 4 * count) + self.endframe

    def setBuffer(self, idx: int, color: tuple):
        r, g, b, a = color
        st = 4*(idx+1)
        en = 4*(idx+2)
        try:
            self._buffer[st:en] = bytearray([0xe0 | a, b, g, r])
        except:
            print(color)

    def fill(self, color: tuple):
        count = self.width * self.height
        for i in range(count):
            self.setBuffer(i, color)

    def pixel(self, x: int, y: int, color: tuple):
        if y % 2:
            x = self.width - 1 - x
        idx = self.width * y + x
        self.setBuffer(idx, color)

    def show(self):
        self._device.send(self._buffer)
