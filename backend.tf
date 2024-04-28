terraform {
  backend "http" {
    address        = "https://ffddorf-terraform-backend.fly.dev/state/terraform-playground/dev"
    lock_address   = "https://ffddorf-terraform-backend.fly.dev/state/terraform-playground/dev"
    unlock_address = "https://ffddorf-terraform-backend.fly.dev/state/terraform-playground/dev"
    username       = "github_pat"
  }
}
