// const (
// 	senderEmail   = "MS_YNKJNo@trial-neqvygm85w840p7w.mlsender.net"
// 	receiverEmail = "msaruni679@gmail.com"
// 	smtpServer    = "smtp.mailersend.net"
// 	smtpPort      = "587"
// 	smtpPassword  = "IY7lWUfg260dD6hR"
// )

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
	"time"

	"github.com/jung-kurt/gofpdf"
	"github.com/skip2/go-qrcode"
)

type GenerateQRRequest struct {
	Email          string `json:"email"`
	EventID        string `json:"event_id"`
	TicketQuantity int    `json:"ticket_quantity"`
}

func main() {
	http.HandleFunc("/generate-qr", handleGenerateQR)
	fmt.Println("Starting HTTP server on port 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handleGenerateQR(w http.ResponseWriter, r *http.Request) {
	var req GenerateQRRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Generate QR codes and send them as separate PDFs
	attachments, err := generateSeparatePDFs(req.Email, req.EventID, req.TicketQuantity)
	if err != nil {
		http.Error(w, "Error generating PDFs", http.StatusInternalServerError)
		return
	}

	// Send the email with the PDFs attached
	if err := sendEmailWithAttachments(req.Email, "QR Codes for Event "+req.EventID, "Attached are your QR codes for the event", attachments); err != nil {
		http.Error(w, "Error sending email", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("QR codes generated and emailed successfully!"))
}

func generateSeparatePDFs(email, eventID string, ticketQuantity int) ([]string, error) {
	var attachments []string

	qrData := fmt.Sprintf("%s%d%s%d", eventID, ticketQuantity, email, time.Now().Unix())
	// Hash the data
	hash := sha256.New()
	hash.Write([]byte(qrData))
	// Convert the hash to a string
	hashStr := hex.EncodeToString(hash.Sum(nil))

	for i := 1; i <= ticketQuantity; i++ {
		pdf := gofpdf.New("P", "mm", "A4", "")
		//  ADD hash function here
		url := fmt.Sprintf("%s-%d", hashStr, i) //URL should be the result if the hash function
		addQRCodeToPDF(pdf, i, url, email, eventID)

		fileName := fmt.Sprintf("%s_ticket_%d.pdf", eventID, i)
		if err := pdf.OutputFileAndClose(fileName); err != nil {
			fmt.Println("Error creating PDF:", err)
			return nil, err
		}
		attachments = append(attachments, fileName)
	}
	return attachments, nil
}

func addQRCodeToPDF(pdf *gofpdf.Fpdf, i int, url string, email string, eventID string) {

	qrCode, err := qrcode.New(url, qrcode.Medium)
	if err != nil {
		fmt.Println("Error generating QR code:", err)
		return
	}

	var qrCodeBuffer bytes.Buffer
	if err := qrCode.Write(256, &qrCodeBuffer); err != nil {
		fmt.Println("Error writing QR code to buffer:", err)
		return
	}

	pdf.AddPage()
	pdf.RegisterImageOptionsReader(fmt.Sprintf("qr%d", i), gofpdf.ImageOptions{ImageType: "png"}, &qrCodeBuffer)

	pdf.ImageOptions(fmt.Sprintf("qr%d", i), 50, 50, 100, 100, false, gofpdf.ImageOptions{ImageType: "png"}, 0, "")

	// Add text below the QR code

	// Add "Your Ticket" button
	pdf.SetFont("Arial", "", 28)
	pdf.SetTextColor(0, 0, 0)
	pdf.SetXY(80, 25)
	pdf.Cell(100, 10, "Your Ticket")

	// Add ticket details
	pdf.SetFont("Arial", "", 16)
	pdf.SetTextColor(0, 0, 0)
	pdf.SetXY(50, 150)
	pdf.Cell(100, 10, fmt.Sprintf("Event: %s", eventID))
	pdf.SetXY(50, 160)
	pdf.Cell(100, 10, fmt.Sprintf("Email: %s", email))

	pdf.SetXY(50, 170)
	pdf.Cell(100, 10, "Ticket: REGULAR")

	pdf.SetXY(50, 180)
	pdf.Cell(100, 10, "Location: KICC, Parliament Road, Nairobi, Kenya")

	pdf.SetXY(50, 190)
	pdf.Cell(100, 10, "Date & Time: N/A")

	pdf.SetFont("Arial", "", 14)
	pdf.SetTextColor(0, 0, 0)
	// pdf.SetXY(25, 200)
	// pdf.Cell(100, 10, "Code")
	pdf.SetXY(25, 205)
	pdf.Cell(50, 10, url)
}

// func sendEmailWithAttachments(receiverEmail, subject, body string, attachments []string) error {
// 	const (
// 		senderEmail  = "MS_YNKJNo@trial-neqvygm85w840p7w.mlsender.net"
// 		smtpServer   = "smtp.mailersend.net"
// 		smtpPort     = "587"
// 		smtpPassword = "IY7lWUfg260dD6hR"
// 	)

// 	auth := smtp.PlainAuth("", senderEmail, smtpPassword, smtpServer)
// 	boundary := fmt.Sprintf("----%x", rand.Int63n(10000000000))

// 	msg := bytes.NewBuffer(nil)
// 	msg.WriteString("MIME-Version: 1.0\r\n")
// 	msg.WriteString(fmt.Sprintf("To: %s\r\n", receiverEmail))
// 	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
// 	msg.WriteString(fmt.Sprintf("Content-Type: multipart/mixed; boundary=%s\r\n", boundary))
// 	msg.WriteString("\r\n--" + boundary + "\r\n")
// 	msg.WriteString("Content-Type: text/plain; charset=utf-8\r\n\r\n" + body + "\r\n")

// 	for _, attachment := range attachments {
// 		if err := addAttachment(msg, attachment, boundary); err != nil {
// 			fmt.Println("Error adding attachment:", err)
// 			return err
// 		}
// 	}

// 	msg.WriteString("\r\n--" + boundary + "--\r\n")

// 	err := smtp.SendMail(smtpServer+":"+smtpPort, auth, senderEmail, []string{receiverEmail}, msg.Bytes())
// 	if err != nil {
// 		return err
// 	}
// 	fmt.Println("Email sent successfully!")
// 	return nil
// }

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
		if err := addAttachment(msg, attachment, boundary); err != nil {
			fmt.Println("Error adding attachment:", err)
			return err
		}
		// Delete the attachment after adding it to the email message
		if err := os.Remove(attachment); err != nil {
			fmt.Printf("Warning: failed to delete file %s: %v\n", attachment, err)
		}
	}

	msg.WriteString("\r\n--" + boundary + "--\r\n")

	err := smtp.SendMail(smtpServer+":"+smtpPort, auth, senderEmail, []string{receiverEmail}, msg.Bytes())
	if err != nil {
		return err
	}
	fmt.Println("Email sent successfully!")
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
