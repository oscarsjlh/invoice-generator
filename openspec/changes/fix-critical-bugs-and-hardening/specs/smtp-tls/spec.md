## ADDED Requirements

### Requirement: SMTP connections must use TLS

The system SHALL configure explicit TLS on the SMTP dialer when sending invoice emails. The TLS configuration SHALL set `ServerName` to the SMTP host to enable certificate verification.

#### Scenario: Email sent with TLS enabled
- **WHEN** an invoice email is sent to a customer
- **THEN** the SMTP connection uses TLS with the server name matching the SMTP host configuration

#### Scenario: TLS certificate verification
- **WHEN** the SMTP server presents a certificate that does not match the configured host
- **THEN** the connection fails and the error is returned to the caller
