import matplotlib.pyplot as plt
import numpy as np

p_values = np.linspace(0.01, 0.5, 100)
R_values = [10, 100, 1000, 10000]

for R in R_values:
    S_values = R * ((1 - p_values) / p_values)
    plt.plot(p_values, S_values, label=f"R = {R} POKT")

plt.xlabel("ProofRequestProbability (p)")
plt.ylabel("Required Penalty (S POKT)")
plt.title("Penalty vs. ProofRequestProbability for Different Reward Values")
plt.legend()
plt.yscale("log")  # Use logarithmic scale for y-axis (optional)
plt.grid(True, which="both", ls="--")
plt.show()
