import matplotlib.pyplot as plt
import numpy as np

# Define different ProofRequestProbability values
p_values = [0.1, 0.25, 0.5, 0.75]  # Modify as needed
k_values = np.arange(0, 21)  # Range of k values starting from 0

# Create subplots side by side
fig, (ax1, ax2) = plt.subplots(1, 2, figsize=(14, 6))

# Plot Geometric PDF for different p values
for p in p_values:
    q = 1 - p
    pdf_values = p * (1 - p) ** k_values
    ax1.plot(k_values, pdf_values, marker="o", linestyle="-", label=f"p = {p}")

ax1.set_title("Geometric PDF: Exactly k Failures Until 1st Success")
ax1.set_xlabel("Number of Failures before First Success (k)")
ax1.set_ylabel("Probability Pr(X = k)")
ax1.set_xticks(k_values)
ax1.grid(True)
ax1.legend()

# Plot Geometric CDF for different p values
for p in p_values:
    q = 1 - p
    cdf_values = 1 - (1 - p) ** (k_values + 1)
    ax2.plot(k_values, cdf_values, marker="o", linestyle="-", label=f"p = {p}")

ax2.set_title("Geometric CDF: ≤ k Failures Until 1st Success")
ax2.set_xlabel("Number of Failures before First Success (k)")
ax2.set_ylabel("Cumulative Probability P(X ≤ k)")
ax2.set_xticks(k_values)
ax2.grid(True)
ax2.legend()

# Adjust layout and display the plots
plt.tight_layout()
plt.show()
