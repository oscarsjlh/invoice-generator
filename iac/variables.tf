variable "aws_region" {
  description = "The AWS region to deploy resources into"
  type        = string
  default     = "us-east-1"
}

variable "project_name" {
  description = "A unique name for the project resources"
  type        = string
}

variable "state_bucket_name" {
  description = "The name of the S3 bucket. Must be globally unique."
  type        = string
}
variable "profile" {
  type = string
}
variable "region" {
  type = string
}

variable "domain_name" {
  type = string
}
