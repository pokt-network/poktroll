import { Client } from "./index.js";

async function main() {
  console.log("Creating client...");
  
  // Query network parameters
  const client = new Client({
    rpcURL: "https://shannon-grove-rpc.mainnet.poktroll.com",
    apiURL: "https://shannon-grove-api.mainnet.poktroll.com",
  });
  
  console.log("Client created successfully!");
  
  try {
    // Get current network status
    console.log("Fetching network status...");
    const status = await client.cosmos.base.tendermint.v1beta1.getNodeInfo();
    console.log("Network status:", status);
    
    // Get latest block
    console.log("Fetching latest block...");
    const latestBlock = await client.cosmos.base.tendermint.v1beta1.getLatestBlock();
    console.log("Latest block height:", latestBlock.block?.header?.height);
    
    // Get all active suppliers
    console.log("Fetching suppliers...");
    const activeSuppliers = await client.pocket.supplier.queryAllSuppliers();
    console.log(`Found ${activeSuppliers.supplier?.length || 0} active suppliers`);
    
    // Get all available services
    console.log("Fetching services...");
    const availableServices = await client.pocket.service.queryAllServices();
    console.log(`Found ${availableServices.service?.length || 0} available services`);
    
  } catch (error) {
    console.error("Error:", error);
  }
}

main().catch(console.error);