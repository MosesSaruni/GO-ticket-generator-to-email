# 📄 2wende QR Code Ticket Generator and Email Sender

## 🎉 About 2wende  
2wende is an innovative ticketing platform dedicated to streamlining event ticket distribution through secure, digital solutions.

---

## ✨ Overview  
This Go-based application is designed to enhance 2wende's ticketing operations by generating QR code tickets, converting them into PDFs, and sending them directly to customers via email.

---

## 💡 Key Features  
- **Seamless QR Code Generation**: Unique QR codes are created for each ticket.  
- **Personalized PDFs**: Each ticket is embedded in a custom PDF.  
- **Email Delivery**: Automatically sends tickets as email attachments.  
- **Data Security**: Utilizes SHA-256 for hashing QR code data.  
- **Temporary Storage**: Cleans up PDF files after emailing.  

---

## 🚀 How to Use  

### 1. Run the Server:  
```bash
go run main.go
```

### 2. Endpoint:  
- **URL**: `http://localhost:8080/generate-qr`  
- **Method**: `POST`  

### 3. Sample Payload:  
```json
{
  "email": "customer@example.com",
  "event_id": "EVENT456",
  "ticket_quantity": 3
}
```

