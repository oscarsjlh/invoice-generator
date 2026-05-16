resource "aws_ses_identity" "ses_identity" {
  name = "${var.project_name}-ses-identity"
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