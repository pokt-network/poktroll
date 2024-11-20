import matplotlib.pyplot as plt
import numpy as np


# Function to calculate Pr(X = k)
def geometric_pmf(k, p):
    return (1 - p) ** (k - 1) * p


# Updated p values to include 1.0
p_values = [0.2, 0.5, 0.8, 1.0]  # Different values for p, including 1.0

# Extend the k_values to start from 0
k_values = np.arange(0, 21)  # k from 0 to 20

plt.figure(figsize=(10, 6))

# Plot the geometric distribution for different p values
for p in p_values:
    probabilities = geometric_pmf(k_values[1:], p)  # Skip k=0 as PMF is undefined for k=0
    plt.plot(k_values[1:], probabilities, marker="o", label=f"p = {p}")

plt.xticks(np.arange(0, 21, 1))  # Set x-axis ticks from 0 to 20
plt.xlabel("k")
plt.ylabel("Pr(X = k)")
plt.title("Geometric Distribution PMF")
plt.legend()
plt.grid(True)
plt.show()
