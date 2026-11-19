package service

import (
	"fmt"

	"payment/internal/mailer"
	"payment/internal/model"
)

func (s *EmailService) message(job model.EmailDelivery, invoice model.Invoice) (mailer.Message, error) {
	if job.InvoiceID != invoice.ID || job.BookingID != invoice.BookingID {
		return mailer.Message{}, mailer.ErrInvalidMessage
	}
	message := mailer.Message{To: job.Recipient, ID: job.MessageID, CreatedAt: job.CreatedAt}
	bookingURL := s.webBaseURL + "/account/bookings/" + invoice.BookingID
	details := fmt.Sprintf("Booking reference: %s\nCar: %s\nPickup: %s WITA\nReturn: %s WITA\nRental: %d day(s) x IDR %d\nTotal paid: IDR %d\n", invoice.BookingID, invoice.VehicleLabel,
		invoice.PickupAt.In(s.wita).Format("02 Jan 2006 15:04"), invoice.ReturnAt.In(s.wita).Format("02 Jan 2006 15:04"), invoice.RentalDays, invoice.DailyRateIDR, invoice.TotalIDR)
	switch job.Kind {
	case "booking_confirmation":
		message.Subject = "SROOM booking confirmed"
		message.Body = "Your booking is confirmed.\n\n" + details + "\nView your booking (sign in required):\n" + bookingURL + "\n\nSROOM sandbox demo.\n"
	case "invoice":
		message.Subject = "SROOM invoice " + invoice.InvoiceNumber
		message.Body = fmt.Sprintf("Your invoice is available.\n\nInvoice number: %s\nNon-tax invoice\n\n%s\nView your invoice (sign in required):\n%s/invoice\n\nSROOM sandbox demo.\n", invoice.InvoiceNumber, details, bookingURL)
	default:
		return mailer.Message{}, mailer.ErrInvalidMessage
	}
	return message, nil
}
