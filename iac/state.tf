terraform {
  backend "s3" {
    bucket  = var.state_bucket_name
    key     = "invoice/state.tfstate"
    region  = var.region
    profile = var.profile
    encrypt = true
  }
}
