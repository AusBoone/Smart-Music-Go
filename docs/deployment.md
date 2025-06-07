# Deployment

This document describes options for deploying Smart-Music-Go in a production environment.

## Build and Push the Docker Image
1. Build the image locally:
   ```bash
   docker build -t smart-music-go .
   ```
2. Tag and push to your registry:
   ```bash
   docker tag smart-music-go <registry>/smart-music-go
   docker push <registry>/smart-music-go
   ```

## Running the Container
Run the image on your platform of choice, passing the environment variables used in development (`SPOTIFY_CLIENT_ID`, `SPOTIFY_CLIENT_SECRET`, `SPOTIFY_REDIRECT_URL`, and `DATABASE_PATH`). When using Docker Compose you can supply an `.env` file.

## AWS Fargate Example
Example Terraform for deploying to AWS Fargate is provided under `deploy/aws`. After pushing your image:
```bash
cd deploy/aws
terraform init
terraform apply
```
Fill in the required variables in `terraform.tfvars` (an example file is included) such as the VPC, subnets, ACM certificate and the Spotify credentials.

The output `load_balancer_dns` contains the HTTPS address for the service.

## Production Configuration
Store secrets using your cloud provider's secret management solution (for example AWS Secrets Manager or Heroku config vars). If you terminate TLS at a load balancer or reverse proxy, ensure that `SPOTIFY_REDIRECT_URL` matches the public HTTPS URL of `/callback`.
