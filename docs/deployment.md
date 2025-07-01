# Deployment

This guide explains how to build and run the project in production. The application ships with a Dockerfile and Terraform for AWS Fargate.

## Build and Push the Docker Image
1. Build locally
   ```bash
   docker build -t smart-music-go .
   ```
2. Push the image to your registry
   ```bash
   docker tag smart-music-go <registry>/smart-music-go:latest
   docker push <registry>/smart-music-go:latest
   ```

## Running the Container
Run the image using your orchestration platform. The following environment variables are required:
- `SPOTIFY_CLIENT_ID`
- `SPOTIFY_CLIENT_SECRET`
- `SPOTIFY_REDIRECT_URL`
- `SIGNING_KEY`
- `DATABASE_PATH` (defaults to `smartmusic.db`)

If using Docker Compose you can supply these via an `.env` file:
```bash
docker compose up --build
```

This starts the Go server on port 8080 and serves the compiled React frontend.
Adjust the compose file if you wish to expose a different port or mount a
persistent volume for the database.

## AWS Fargate Walkthrough
An example Terraform configuration resides under `deploy/aws`. After pushing your image:
```bash
cd deploy/aws
cp terraform.tfvars.example terraform.tfvars
# edit terraform.tfvars and provide Spotify credentials, VPC and subnet IDs
terraform init
terraform apply
```
Terraform provisions an ECS service behind an Application Load Balancer. Once complete, `load_balancer_dns` in the output lists the HTTPS endpoint. Set `SPOTIFY_REDIRECT_URL` to `https://<load_balancer_dns>/callback`.

To tear the stack down run:
```bash
terraform destroy
```
Review the generated plan before confirming deletion.

## Production Configuration
Store secrets using your cloud provider's secret manager. When terminating TLS at a load balancer or reverse proxy ensure the redirect URL matches the public HTTPS address.
