terraform {
  backend "http" {
    address        = "https://ffddorf-terraform-backend.fly.dev/state/playground/dev"
    lock_address   = "https://ffddorf-terraform-backend.fly.dev/state/playground/dev"
    unlock_address = "https://ffddorf-terraform-backend.fly.dev/state/playground/dev"
    username       = "github_pat"
  }
}
