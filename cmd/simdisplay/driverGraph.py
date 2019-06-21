import matplotlib.pyplot as plt
import numpy as np
import json

data = []

with open('./driverOutput') as f:
    for line in f.readlines():
        data.append( json.loads(line)['amp'] )


data = np.array(data)

print(data)
y = np.arange(len(data))

for i in range(len(data[0])):
    plt.plot(data[:, i])

plt.show()