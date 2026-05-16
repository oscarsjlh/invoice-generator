resource "aws_ses_domain_identity" "ses_identity" {
  domain = var.domain_name
}

resource "aws_iam_policy" "ses_iam_policy" {
  name        = "${var.project_name}-ses-policy"
  description = "Policy granting SES sending permissions for ${var.project_name}"

  policy = jsonencode({
    Version = "2012-10-17",
    Statement = [
      {
        Effect = "Allow",
        Action = [
          "ses:SendEmail",
          "ses:SendRawEmail",
          "ses:SendBulkEmail",
        ],
        Resource = "*"
      }
    ]
  })
}

resource "aws_iam_user" "ses_user" {
  name = "${var.project_name}-ses-user"
}

resource "aws_iam_user_policy_attachment" "ses_user_attach" {
  user       = aws_iam_user.ses_user.name
  policy_arn = aws_iam_policy.ses_iam_policy.arn
}

data "aws_route53_zone" "main" {
  name         = var.domain_name
  private_zone = false
}

################################################################################
# SES Domain Identity
################################################################################

resource "aws_ses_domain_identity" "main" {
  domain = var.domain_name
}

################################################################################
# SES Verification TXT Record
################################################################################

resource "aws_route53_record" "ses_verification" {
  zone_id = data.aws_route53_zone.main.zone_id
  name    = "_amazonses.${var.domain_name}"
  type    = "TXT"
  ttl     = 300

  records = [
    aws_ses_domain_identity.main.verification_token
  ]
}

################################################################################
# Wait for SES Verification
################################################################################

resource "aws_ses_domain_identity_verification" "main" {
  domain = aws_ses_domain_identity.main.id

  depends_on = [
    aws_route53_record.ses_verification
  ]
}

################################################################################
# DKIM
################################################################################

resource "aws_ses_domain_dkim" "main" {
  domain = aws_ses_domain_identity.main.domain
}

resource "aws_route53_record" "dkim" {
  count   = 3
  zone_id = data.aws_route53_zone.main.zone_id

  name = "${aws_ses_domain_dkim.main.dkim_tokens[count.index]}._domainkey.${var.domain_name}"
  type = "CNAME"
  ttl  = 300

  records = [
    "${aws_ses_domain_dkim.main.dkim_tokens[count.index]}.dkim.amazonses.com"
  ]
}

################################################################################
# SPF
################################################################################

resource "aws_route53_record" "spf" {
  zone_id = data.aws_route53_zone.main.zone_id
  name    = var.domain_name
  type    = "TXT"
  ttl     = 300

  records = [
    "v=spf1 include:amazonses.com ~all"
  ]
}

################################################################################
# DMARC
################################################################################

resource "aws_route53_record" "dmarc" {
  zone_id = data.aws_route53_zone.main.zone_id
  name    = "_dmarc.${var.domain_name}"
  type    = "TXT"
  ttl     = 300

  records = [
    "v=DMARC1; p=quarantine; pct=100; fo=1;"
  ]
}

################################################################################
# Optional MAIL FROM domain
################################################################################

resource "aws_ses_domain_mail_from" "main" {
  domain           = aws_ses_domain_identity.main.domain
  mail_from_domain = "mail.${var.domain_name}"
}

resource "aws_route53_record" "mail_from_mx" {
  zone_id = data.aws_route53_zone.main.zone_id
  name    = aws_ses_domain_mail_from.main.mail_from_domain
  type    = "MX"
  ttl     = 300

  records = [
    "10 feedback-smtp.eu-west-2.amazonses.com"
  ]
}

resource "aws_route53_record" "mail_from_spf" {
  zone_id = data.aws_route53_zone.main.zone_id
  name    = aws_ses_domain_mail_from.main.mail_from_domain
  type    = "TXT"
  ttl     = 300

  records = [
    "v=spf1 include:amazonses.com ~all"
  ]
}
