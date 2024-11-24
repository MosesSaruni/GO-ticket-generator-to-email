📄 2wende QR Code Ticket Generator and Email Sender
🎉 About 2wende
2wende is an innovative ticketing platform dedicated to streamlining event ticket distribution through secure, digital solutions.

✨ Overview
This Go-based application is designed to enhance 2wende's ticketing operations by generating QR code tickets, converting them into PDFs, and sending them directly to customers via email.

💡 Key Features
Seamless QR Code Generation: Unique QR codes are created for each ticket.
Personalized PDFs: Each ticket is embedded in a custom PDF.
Email Delivery: Automatically sends tickets as email attachments.
Data Security: Utilizes SHA-256 for hashing QR code data.
Temporary Storage: Cleans up PDF files after emailing.
🚀 How to Use
Run the Server:
bash
Copy code
go run main.go
Endpoint:
URL: http://localhost:8080/generate-qr
Method: POST
Sample Payload:
json
Copy code
{
  "email": "customer@example.com",
  "event_id": "EVENT456",
  "ticket_quantity": 3
}
📧 Email Sample
Subject: QR Codes for Event EVENT456
Attachments: Files named like EVENT456_ticket_1.pdf, EVENT456_ticket_2.pdf, etc.
⚙️ Implementation Details
Dependencies:

gofpdf: For PDF generation.
go-qrcode: For QR code generation.
crypto/sha256, net/http, net/smtp: Standard Go libraries for hashing, server, and email.
Functions:

handleGenerateQR: Processes incoming requests and triggers ticket creation.
generateSeparatePDFs: Creates individual PDF files for each ticket.
addQRCodeToPDF: Customizes the PDF with QR code and event details.
sendEmailWithAttachments: Sends the generated PDF tickets via email.
🛡️ Security Best Practices
Replace the hardcoded email credentials with environment variables for better security.
🌱 Future Upgrades for 2wende
Customizable ticket layouts.
Support for additional languages and currencies.
Integration with event management tools.
🌐 Contact Us
For any assistance or feedback, reach out to 2wende Support at support@2wende.com.
