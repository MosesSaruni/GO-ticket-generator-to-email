package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/smtp"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/jung-kurt/gofpdf"
	"github.com/skip2/go-qrcode"
	"github.com/supabase-community/supabase-go"
)

type GenerateQRRequest struct {
	Email          string `json:"email"`
	EventID        string `json:"event_id"`
	TicketQuantity int    `json:"ticket_quantity"`
	Venue          string `json:"venue"`
	Date           string `json:"date"`
	Event_name     string `json:"event_name"`
	User_name      string `json:"user_name"`
}

// Supabase client as global for reuse
var supabaseClient *supabase.Client

func init() {
	// Load environment variables
	// err := godotenv.Load(".env")
	// if err != nil {
	// 	log.Fatalf("Error loading .env file: %v", err)
	// }

	// Initialize Supabase client once
	supabaseUrl := os.Getenv("SUPABASE_URL")
	supabaseAnonKey := os.Getenv("SUPABASE_ANON_KEY")
	if supabaseUrl == "" || supabaseAnonKey == "" {
		log.Fatalf("supabase URL or anon key not set")
	}
	client, err := supabase.NewClient(supabaseUrl, supabaseAnonKey, &supabase.ClientOptions{})
	if err != nil {
		log.Fatalf("Error initializing Supabase client: %v", err)
	}
	supabaseClient = client
}

func main() {
	http.HandleFunc("/generate-qr", handleGenerateQR)
	fmt.Println("Starting HTTP server on port 8080")
	log.Fatal(http.ListenAndServe(":8080", nil)) // Go handles requests concurrently by default
}

func addTicketData(event_id, transaction_id, ticketCode string) error {
	// Prepare ticket data for insertion
	ticketData := map[string]interface{}{
		"ticket_id":      ticketCode,
		"transaction_id": transaction_id,
		"scanned":        false,
		"event_id":       event_id,
	}

	// Insert the ticket data into the "TICKETS" table
	_, _, err := supabaseClient.From("TICKETS").Insert(ticketData, false, "", "*", "").Execute()
	if err != nil {
		log.Printf("Detailed Supabase client error: %v", err)
		return fmt.Errorf("error adding ticket data: %w", err)
	}
	return nil
}

func addTransactionData(code string, quantity int, email string) error {
	// Prepare transaction data for insertion
	transactionData := map[string]interface{}{
		"transaction_id": code,
		"purchased_by":   email,
		"quantity":       quantity,
		"delivered":      true,
	}

	// Insert the transaction data into the "TRANSACTIONS" table
	_, _, err := supabaseClient.From("TRANSACTIONS").Insert(transactionData, false, "", "*", "").Execute()
	if err != nil {
		log.Printf("Detailed Supabase client error: %v", err)
		return fmt.Errorf("error adding transaction data: %w", err)
	}
	return nil
}

func handleGenerateQR(w http.ResponseWriter, r *http.Request) {
	// Allow cross-origin requests
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	// Handle preflight request (OPTIONS)
	if r.Method == http.MethodOptions {
		return
	}

	var req GenerateQRRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Process the request concurrently using a goroutine
	errorChan := make(chan error, 1)
	go func() {
		attachments, err := generateSeparatePDFs(req.Email, req.Event_name, req.EventID, req.TicketQuantity, req.Venue, req.Date)
		if err != nil {
			errorChan <- fmt.Errorf("error generating PDFs: %w", err)
			return
		}

		if err := sendEmailWithAttachments(req.Email, "Your tickets are here! ", "Hey "+req.User_name+" ! \n"+"Here's your ticket for "+req.Event_name+"\n Enjoy!", attachments); err != nil {
			errorChan <- fmt.Errorf("error sending email: %w", err)
			return
		}

		errorChan <- nil // No error, operation successful
	}()

	// Wait for goroutine to complete
	if err := <-errorChan; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("QR codes generated and emailed successfully!"))
}

func generateSeparatePDFs(email, eventName string, eventID string, ticketQuantity int, venue string, date string) ([]string, error) {
	var wg sync.WaitGroup
	attachments := make([]string, 0, ticketQuantity)

	hash := sha256.New()
	hash.Write([]byte(fmt.Sprintf("%s%d%s%d", eventID, ticketQuantity, email, time.Now().Unix())))
	hashStr := hex.EncodeToString(hash.Sum(nil))

	// Insert transaction data once, before starting the PDF generation
	if err := addTransactionData(hashStr, ticketQuantity, email); err != nil {
		return nil, fmt.Errorf("error adding transaction data: %w", err)
	}

	// Generate PDFs concurrently
	for i := 1; i <= ticketQuantity; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			pdf := gofpdf.New("P", "mm", "A4", "")
			url := fmt.Sprintf("%s-%d", hashStr, i)
			addQRCodeToPDF(pdf, i, url, email, eventName, venue, date)

			// Specifying output directory for PDFs#
			outPutDir := "/src/output"
			fileName := fmt.Sprintf("%s_ticket_%d.pdf", eventID, i)
			fullPath := filepath.Join(outPutDir, fileName)

			if err := pdf.OutputFileAndClose(fullPath); err != nil {
				log.Printf("Error creating PDF: %v", err)
				return
			}

			if err := addTicketData(eventID, hashStr, url); err != nil {
				log.Printf("Error adding ticket data: %v", err)
				return
			}

			attachments = append(attachments, fileName)
		}(i)
	}

	wg.Wait()

	return attachments, nil
}

func addQRCodeToPDF(pdf *gofpdf.Fpdf, i int, url string, email string, eventName string, venue string, date string) {
	qrCode, err := qrcode.New(url, qrcode.Medium)
	if err != nil {
		log.Println("Error generating QR code:", err)
		return
	}

	var qrCodeBuffer bytes.Buffer
	if err := qrCode.Write(256, &qrCodeBuffer); err != nil {
		log.Println("Error writing QR code to buffer:", err)
		return
	}

	pdf.AddPage()

	// Draw a header background
	pdf.SetFillColor(40, 12, 166)  // Cornflower blue for header background
	pdf.Rect(10, 10, 190, 30, "F") // Full-width rectangle for the header

	// Set title with bold, large font and centered
	pdf.SetFont("Arial", "B", 32)
	pdf.SetTextColor(255, 255, 255) // White for title text
	pdf.SetXY(10, 15)
	pdf.CellFormat(190, 25, "2ende", "", 0, "C", false, 0, "")

	// Add the QR code with a smaller size and rounded border
	pdf.RegisterImageOptionsReader(fmt.Sprintf("qr%d", i), gofpdf.ImageOptions{ImageType: "png"}, &qrCodeBuffer)
	pdf.SetDrawColor(100, 149, 237) // Border color matches header
	pdf.SetLineWidth(1)
	pdf.RoundedRect(15, 50, 70, 70, 5, "D", "1234") // Rounded border for the QR code
	pdf.ImageOptions(fmt.Sprintf("qr%d", i), 20, 55, 60, 60, false, gofpdf.ImageOptions{ImageType: "png"}, 0, "")

	// Draw an information section border of the same size as the QR code
	pdf.SetDrawColor(100, 149, 237) // Matching border color
	pdf.SetLineWidth(1)
	pdf.RoundedRect(90, 50, 110, 70, 5, "D", "1234") // Rounded border for ticket details

	// Ticket details with improved padding and font styles
	pdf.SetFont("Arial", "", 16)
	pdf.SetTextColor(50, 50, 50) // Dark gray for text

	pdf.SetXY(95, 55)
	pdf.Cell(60, 10, fmt.Sprintf("Event: %s", eventName))

	pdf.SetXY(95, 65)
	pdf.Cell(60, 10, fmt.Sprintf("Email: %s", email))

	pdf.SetXY(95, 75)
	pdf.Cell(60, 10, "Ticket: REGULAR")

	pdf.SetXY(95, 85)
	pdf.Cell(60, 10, fmt.Sprintf("Location: %s", venue))

	pdf.SetXY(95, 95)
	pdf.Cell(60, 10, fmt.Sprintf("Date: %s", date))

	// Add the URL with a modern link style
	pdf.SetFont("Arial", "I", 14)
	pdf.SetTextColor(11, 135, 34) // Blue for the URL
	pdf.SetXY(15, 130)
	pdf.CellFormat(180, 10, url, "", 0, "C", false, 0, "")

	// Add a footer line with a modern touch
	pdf.SetDrawColor(230, 230, 230) // Light gray
	pdf.SetLineWidth(0.5)
	pdf.Line(10, 145, 200, 145) // Thin, decorative line for footer

}

func sendEmailWithAttachments(receiverEmail, subject, body string, attachments []string) error {
	const (
		senderEmail  = "MS_YNKJNo@trial-neqvygm85w840p7w.mlsender.net"
		smtpServer   = "smtp.mailersend.net"
		smtpPort     = "587"
		smtpPassword = "IY7lWUfg260dD6hR"
	)

	auth := smtp.PlainAuth("", senderEmail, smtpPassword, smtpServer)
	boundary := fmt.Sprintf("----%x", rand.Int63n(10000000000))

	msg := bytes.NewBuffer(nil)
	msg.WriteString("MIME-Version: 1.0\r\n")
	msg.WriteString(fmt.Sprintf("To: %s\r\n", receiverEmail))
	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	msg.WriteString(fmt.Sprintf("Content-Type: multipart/mixed; boundary=%s\r\n", boundary))
	msg.WriteString("\r\n--" + boundary + "\r\n")
	msg.WriteString("Content-Type: text/plain; charset=utf-8\r\n\r\n" + body + "\r\n")

	for _, attachment := range attachments {
		attachmentPath := filepath.Join("/src/output", attachment)
		if err := addAttachment(msg, attachmentPath, boundary); err != nil {
			log.Printf("Error adding attachment: %v", err)
			return err
		}
		if err := os.Remove(attachmentPath); err != nil {
			log.Printf("Warning: failed to delete file %s: %v", attachment, err)
		}
	}

	msg.WriteString("\r\n--" + boundary + "--\r\n")
	err := smtp.SendMail(smtpServer+":"+smtpPort, auth, senderEmail, []string{receiverEmail}, msg.Bytes())
	if err != nil {
		log.Printf("Error sending email: %v", err)
		return err
	}
	log.Println("Email sent successfully!")
	return nil
}

func addAttachment(msg *bytes.Buffer, attachmentPath, boundary string) error {
	file, err := os.Open(attachmentPath)
	if err != nil {
		return fmt.Errorf("error opening attachment: %w", err)
	}
	defer file.Close()

	_, fileName := filepath.Split(attachmentPath)
	msg.WriteString("\r\n--" + boundary + "\r\n")
	msg.WriteString(fmt.Sprintf("Content-Type: application/pdf; name=\"%s\"\r\n", fileName))
	msg.WriteString("Content-Transfer-Encoding: base64\r\n")
	msg.WriteString(fmt.Sprintf("Content-Disposition: attachment; filename=\"%s\"\r\n", fileName))
	msg.WriteString("\r\n")

	buffer := make([]byte, 1024)
	b64Encoder := base64.NewEncoder(base64.StdEncoding, msg)
	defer b64Encoder.Close()

	for {
		n, err := file.Read(buffer)
		if err == io.EOF {
			break
		} else if err != nil {
			return fmt.Errorf("error reading attachment: %w", err)
		}
		if _, err := b64Encoder.Write(buffer[:n]); err != nil {
			return fmt.Errorf("error encoding attachment: %w", err)
		}
	}
	return nil
}
