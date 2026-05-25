ALTER TABLE invoices ADD COLUMN customer_name TEXT NOT NULL DEFAULT '';
ALTER TABLE invoices ADD COLUMN customer_title TEXT NOT NULL DEFAULT '';
ALTER TABLE invoices ADD COLUMN customer_email TEXT NOT NULL DEFAULT '';
ALTER TABLE invoices ADD COLUMN customer_address TEXT NOT NULL DEFAULT '';
ALTER TABLE invoices ADD COLUMN customer_postal_code TEXT NOT NULL DEFAULT '';
ALTER TABLE invoices ADD COLUMN customer_city TEXT NOT NULL DEFAULT '';
