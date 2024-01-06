# Terraform Example

This is a Terraform setup running with this configuration:
- State is stored in [tfstate.dev](https://tfstate.dev)
- Plans and applies run in GitHub Actions
- State is backed up to Backblaze B2

## Local Preview

This setup includes a tool to run speculatives plans for local changes in GitHub Actions so that secrets can be used while planning. This setup is supposed to mimic speculative plans in Terraform Cloud.

Prerequisites:
- [ngrok](https://ngrok.com/) Account
  - Put your [auth token](https://dashboard.ngrok.com/get-started/your-authtoken) into the `NGROK_AUTHTOKEN` env var
- GitHub CLI authenticated
- Go installed (no binaries yet)

With that set you can run a speculative plan:

```sh
go run ./cmd/tf-preview-gh
```
