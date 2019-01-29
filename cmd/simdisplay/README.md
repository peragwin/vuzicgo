# Installation on Raspberry PI

1. Set up i2s audio input
https://learn.adafruit.com/adafruit-i2s-mems-microphone-breakout/raspberry-pi-wiring-and-test

2. Install golang (use golang 1.11.1)
https://gist.github.com/random-robbie/1f7f94beda1221b8125b62abe35f45b6

3. Install dependencies
```
apt install portaudio19-dev xorg-dev libgl1-mesa-dev
```

4. Run
```
./simdisplay -mode 1 -buckets 60 -columns 120 -pilocal -headless -frame-rate 100
```

5. Install systemd unit
```
cp ./led-display.service /etc/systemd/system
```