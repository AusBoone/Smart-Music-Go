# AWS Terraform Example

This directory contains Terraform configuration for deploying Smart-Music-Go to AWS.
The variables `spotify_client_id`, `spotify_client_secret` and `spotify_redirect_url` hold
Spotify credentials and are marked as `sensitive` in `variables.tf`. Provide these values
using `terraform.tfvars` or `-var` command line flags and avoid committing them to version control.
