import { ethers } from "ethers";

export const CONFIG = {
  PORT: process.env.PORT || 3000,
  OPENROUTER_API_KEY: process.env.OPENROUTER_API_KEY || "",
  OPENROUTER_MODEL: process.env.OPENROUTER_MODEL || "z-ai/glm-4.5-air:free",

  // Server wallet private key for signing/facilitating (if needed) or just identifying the recipient
  SERVER_PRIVATE_KEY: process.env.SERVER_WALLET_PRIVATE_KEY || "",
  CHAIN_ID: parseInt(process.env.CHAIN_ID || "8453"), // Base

  // Payment details
  PAYMENT: {
    TOKEN_SYMBOL: "USDC",
    // USDC contract address - defaults to Base USDC if not specified
    TOKEN_ADDRESS: process.env.USDC_TOKEN_ADDRESS || "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913", //dummy address btw :)
    DEFAULT_PRICE: process.env.PAYMENT_AMOUNT || "0.001", //dummy
    RECIPIENT_ADDRESS: "", // Will be derived from private key
  }
};

// Derive recipient address from private key if available
if (CONFIG.SERVER_PRIVATE_KEY) {
  const wallet = new ethers.Wallet(CONFIG.SERVER_PRIVATE_KEY);
  CONFIG.PAYMENT.RECIPIENT_ADDRESS = wallet.address;
} else {
  console.warn("No SERVER_WALLET_PRIVATE_KEY provided. Payment verification might fail if it relies on matching recipient.");
}
