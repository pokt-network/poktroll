import numpy as np
import matplotlib.pyplot as plt
from matplotlib import cm

def pdf(x, p):
    return np.log(x/p) / np.log(1-p)

# Line Graph
x = np.linspace(0.01, 1, 200)

# Points
xp = np.linspace(0.01, 1, 20)

# Plot the actual functions
ps = [0.25, 0.5, 0.75, 0.9]
colors = cm.get_cmap('hsv', len(ps)+1)
for i, p in enumerate(ps):
    color = colors(i)
    y = pdf(x, p)
    yp = pdf(xp, p)
    plt.plot(x, y, label=f'p = {p}', color=color)
    # Select only the points where y > 0 and plot them as dots
    x_pos = xp[np.where(yp > 0)]
    y_pos = yp[np.where(yp > 0)]
    plt.plot(x_pos, y_pos, 'o', color=color)


# Add a horizontal line at y = 0
plt.axhline(y=0, color='gray', linestyle='--')

# Add legend, axis labels, and title
plt.legend()
plt.xlabel('Probability(X=k)')
plt.ylabel('k (num failures)')
plt.title('Number of failures until a single success')

# Display the plot
plt.show()