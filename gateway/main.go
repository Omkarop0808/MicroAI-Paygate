package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
)

type PaymentContext struct {
	Recipient string `json:"recipient"`
	Token     string `json:"token"`
	Amount    string `json:"amount"`
	Nonce     string `json:"nonce"`
	ChainID   int    `json:"chainId"`
}

type VerifyRequest struct {
	Context   PaymentContext `json:"context"`
	Signature string         `json:"signature"`
}

type VerifyResponse struct {
	IsValid          bool   `json:"is_valid"`
	RecoveredAddress string `json:"recovered_address"`
	Error            string `json:"error"`
}

type SummarizeRequest struct {
	Text string `json:"text"`
}

func main() {
	err := godotenv.Load("../.env")
	if err != nil {
		log.Println("Warning: Error loading .env file")
	}

	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3001"},
		AllowMethods:     []string{"GET", "POST", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "X-402-Signature", "X-402-Nonce"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	r.GET("/healthz", handleHealth)
	r.POST("/api/ai/summarize", handleSummarize)

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	log.Printf("Go Gateway running on port %s", port)
	r.Run(":" + port)
}

func handleSummarize(c *gin.Context) {
	signature := c.GetHeader("X-402-Signature")
	nonce := c.GetHeader("X-402-Nonce")

	// 1. Payment Required
	if signature == "" || nonce == "" {
		context := createPaymentContext()
		c.JSON(402, gin.H{
			"error":          "Payment Required",
			"message":        "Please sign the payment context",
			"paymentContext": context,
		})
		return
	}

	// 2. Verify Payment (Call Rust Service)
	context := PaymentContext{
		Recipient: getRecipientAddress(),
		Token:     "USDC",
		Amount:    "0.001",
		Nonce:     nonce,
		ChainID:   8453,
	}

	verifyReq := VerifyRequest{
		Context:   context,
		Signature: signature,
	}

	verifyBody, _ := json.Marshal(verifyReq)
	verifierURL := os.Getenv("VERIFIER_URL")
	if verifierURL == "" {
		verifierURL = "http://127.0.0.1:3002"
	}
	resp, err := http.Post(verifierURL+"/verify", "application/json", bytes.NewBuffer(verifyBody))
	if err != nil {
		c.JSON(500, gin.H{"error": "Verification service unavailable"})
		return
	}
	defer resp.Body.Close()

	var verifyResp VerifyResponse
	json.NewDecoder(resp.Body).Decode(&verifyResp)

	if !verifyResp.IsValid {
		c.JSON(403, gin.H{"error": "Invalid Signature", "details": verifyResp.Error})
		return
	}

	// 3. Call AI Service
	var req SummarizeRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request body"})
		return
	}

	summary, err := callOpenRouter(req.Text)
	if err != nil {
		c.JSON(500, gin.H{"error": "AI Service Failed", "details": err.Error()})
		return
	}

	c.JSON(200, gin.H{"result": summary})
}

func createPaymentContext() PaymentContext {
	return PaymentContext{
		Recipient: getRecipientAddress(),
		Token:     "USDC",
		Amount:    "0.001",
		Nonce:     uuid.New().String(),
		ChainID:   8453,
	}
}

func getRecipientAddress() string {
	addr := os.Getenv("RECIPIENT_ADDRESS")
	if addr == "" {
		log.Println("Warning: RECIPIENT_ADDRESS not set, using default")
		return "0x2cAF48b4BA1C58721a85dFADa5aC01C2DFa62219"
	}
	return addr
}

func callOpenRouter(text string) (string, error) {
	apiKey := os.Getenv("OPENROUTER_API_KEY")
	model := os.Getenv("OPENROUTER_MODEL")
	if model == "" {
		model = "z-ai/glm-4.5-air:free"
	}

	prompt := fmt.Sprintf("Summarize this text in 2 sentences: %s", text)

	reqBody, _ := json.Marshal(map[string]interface{}{
		"model": model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	})

	req, _ := http.NewRequest("POST", "https://openrouter.ai/api/v1/chat/completions", bytes.NewBuffer(reqBody))
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	if choices, ok := result["choices"].([]interface{}); ok && len(choices) > 0 {
		choice := choices[0].(map[string]interface{})
		message := choice["message"].(map[string]interface{})
		return message["content"].(string), nil
	}

	return "", fmt.Errorf("invalid response from AI provider")
}

func handleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "gateway"})
}
