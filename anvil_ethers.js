const { ethers } = require("ethers");

const provider = new ethers.JsonRpcProvider("http://localhost:8547");

// Default Anvil private key (1st account)
const privateKey =
  "0x59c6995e998f97a5a0044976f9d4b4c2b6e6dcd0bdfbff6af5b7d4b8d7c5c6d0";
const wallet = new ethers.Wallet(privateKey, provider);

console.log("Using wallet address:", wallet.address);

async function main() {
  const largeData = "0x" + "deadbeef".repeat(25000); // ~10KB
  const txs = [];

  console.log("Sending 100 large transactions...");

  const baseNonce = await provider.getTransactionCount(
    wallet.address,
    "latest"
  );
  const baseGasPrice = await provider.getFeeData().then((f) => f.gasPrice);

  for (let i = 0; i < 100; i++) {
    const uniqueData = largeData + i.toString(16).padStart(2, "0");

    const tx = await wallet.sendTransaction({
      to: wallet.address,
      value: 0,
      data: uniqueData,
      gasLimit: 10_000_000n,
      nonce: baseNonce + i,
      gasPrice: baseGasPrice + BigInt(i * 1_000),
    });

    txs.push(tx);
  }

  console.log("Waiting for confirmations...");
  const receipts = await Promise.all(txs.map((tx) => tx.wait()));

  const blockHeights = [...new Set(receipts.map((r) => r.blockNumber))];

  console.log("✅ All transactions confirmed.");
  console.log(
    "📦 Transactions were mined at block height(s):",
    blockHeights.map((n) => "0x" + n.toString(16)).join(", ")
  );
}

main().catch(console.error);
