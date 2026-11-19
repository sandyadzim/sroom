BEGIN;
DROP TABLE payment.inbox_events;
DROP TABLE payment.invoices;
ALTER TABLE payment.payment_attempts DROP COLUMN customer_email, DROP COLUMN customer_full_name;
COMMIT;
